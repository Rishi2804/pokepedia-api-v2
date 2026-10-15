#!/usr/bin/env bash
# Downloads item images into the MinIO seed folder as items/<slug>.png,
# matching the `sprite` key stored on every row of public.item.
#
# Larger images win. Bulbapedia's bag and Dream World images are 120-160px,
# PokeAPI's sprites are ~30px pixel art, so Bulbapedia is tried first and
# PokeAPI fills in only where Bulbapedia has nothing larger:
#   1. Bulbapedia, for every item: the lead image of the item's page, but only
#      when it is a bag or Dream World image ("Bag <Name> ... Sprite.png",
#      "Dream <Name> Sprite.png"); lead images that are menu screenshots or
#      artwork are skipped, because a wrong picture is worse than a small one.
#      It replaces an image already on disk only if it is wider, so a rerun
#      upgrades old small sprites in place.
#   2. The PokeAPI sprites repo, for every item still without an image.
#   3. Bulbapedia again, for every item still without a large image, whose page
#      has no usable lead image: the widest "Bag <Name> <game> Sprite.png" or
#      "Dream <Name> Sprite.png" file that exists. Items it finds nothing
#      larger for are remembered in seed/.item-images-checked so reruns skip
#      them (delete that file to look again).
#   4. One generic Dynamax Crystal image, shared by the 300 star-named
#      Dynamax Crystals, which have no image of their own.
# Anything still missing is counted at the end; the UI shows a placeholder
# icon for those, so a miss is cosmetic.
#
# Needs the pokepedia-db container running with the item migrations applied;
# override DB_CONTAINER / DB_USER / DB_NAME if yours differ. Only reads the
# database.
#
# Idempotent: files already on disk are skipped, so it is safe to re-run.
# After running, `docker compose up minio-init` uploads whatever is new.
set -euo pipefail

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="$HERE/seed/items"
POKEAPI_URL="https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/items"
BULBA_API="https://bulbapedia.bulbagarden.net/w/api.php"
UA="PokePediaScrapeTool/1.0 (+https://github.com/Rishi2804/pokepedia-api-v2)"
PARALLEL="${PARALLEL:-8}"
DB_CONTAINER="${DB_CONTAINER:-pokepedia-db}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-pokepediadb}"
TITLES_PER_REQUEST=20
# Bag images are 30-160px and well under 100KB; anything bigger is a wrong
# page image (a 1080p screenshot slipped through once), so refuse it.
MAX_IMAGE_BYTES=262144
# At or below this width an image is small pixel art that a Bulbapedia bag
# image should replace.
SMALL_MAX_WIDTH=64

# Bulbapedia's bag image game suffixes, newest game first.
BAG_GAMES=(ZA SV SwSh LA BDSP USUM SM ORAS XY B2W2 BW HGSS Pt DP E FRLG RS C GS)

mkdir -p "$DEST"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# slug<TAB>display name<TAB>category<TAB>holdable for every item.
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -At -F $'\t' \
  -c "SELECT name, display_name, category, holdable FROM item ORDER BY id" > "$WORK/items.tsv"
[ -s "$WORK/items.tsv" ] || { echo "no items in $DB_NAME -- apply the item migrations first" >&2; exit 1; }
echo "$(wc -l < "$WORK/items.tsv" | tr -d ' ') items"

count_images() { find "$DEST" -name '*.png' | wc -l | tr -d ' '; }

# Width in pixels of a PNG, from its IHDR chunk; 0 if unreadable.
png_width() {
  local hex
  hex="$(xxd -s 16 -l 4 -p "$1" 2>/dev/null)" || { echo 0; return; }
  [ -n "$hex" ] && printf '%d\n' "0x$hex" || echo 0
}

bulba_get() {
  curl -sf -G -A "$UA" --retry 3 "$BULBA_API" \
    --data-urlencode "format=json" --data-urlencode "formatversion=2" "$@"
}

# The Bulbapedia page title for an item: the display name, minus the "LA "
# prefix on the Legends: Arceus balls (Bulbapedia has no such page). The
# Gen 2 cure berries are titled without spaces, and "Berry" needs "(item)".
page_title() {
  local title="${1#LA }"
  case "$title" in
    "PRZ Cure Berry") title="PRZCureBerry" ;;
    "PSN Cure Berry") title="PSNCureBerry" ;;
    "Berry") title="Berry (item)" ;; # plain "Berry" is the concept page
  esac
  printf '%s' "$title"
}

# --- 1. Bulbapedia lead images, bag images only ------------------------
# slug<TAB>title for every item with no image yet or only a small one.
: > "$WORK/todo.tsv"
while IFS=$'\t' read -r name display _category _holdable; do
  if [ ! -s "$DEST/$name.png" ] || [ "$(png_width "$DEST/$name.png")" -le "$SMALL_MAX_WIDTH" ]; then
    printf '%s\t%s\n' "$name" "$(page_title "$display")" >> "$WORK/todo.tsv"
  fi
done < "$WORK/items.tsv"

# Prints "title<TAB>url" for each requested title whose page has a lead image.
# Requested titles are mapped through MediaWiki's normalization and redirects.
lead_images() {
  local titles_json="$1"
  bulba_get --data-urlencode "action=query" --data-urlencode "prop=pageimages" \
    --data-urlencode "piprop=original" --data-urlencode "redirects=1" \
    --data-urlencode "titles=$(jq -r 'join("|")' <<<"$titles_json")" \
    | jq -r --argjson req "$titles_json" '
        (.query.normalized // []) as $n | (.query.redirects // []) as $r
        | (reduce .query.pages[] as $p ({}; if $p.original then . + {($p.title): $p.original.source} else . end)) as $img
        | $req[] | . as $t
        | ([$n[] | select(.from == $t) | .to][0] // $t) as $a
        | ([$r[] | select(.from == $a) | .to][0] // $a) as $f
        | select($img[$f]) | [$t, $img[$f]] | @tsv'
}

: > "$WORK/leads.tsv"
cut -f2 "$WORK/todo.tsv" | sort -u > "$WORK/titles.txt"
total_titles=$(wc -l < "$WORK/titles.txt" | tr -d ' ')
for ((i = 1; i <= total_titles; i += TITLES_PER_REQUEST)); do
  chunk="$(sed -n "${i},$((i + TITLES_PER_REQUEST - 1))p" "$WORK/titles.txt" | jq -R . | jq -s .)"
  lead_images "$chunk" >> "$WORK/leads.tsv" || true
  sleep 0.4 # polite: one batched request per 20 titles
done

# Keep only bag/Dream images, then download for every slug that title serves.
grep -E $'\t.*/(Bag|Dream)_[^/]*_Sprite\\.png$' "$WORK/leads.tsv" > "$WORK/bag_leads.tsv" || true
skipped=$(( $(wc -l < "$WORK/leads.tsv") - $(wc -l < "$WORK/bag_leads.tsv") ))

filled=0
upgraded=0
while IFS=$'\t' read -r name title; do
  url="$(awk -F'\t' -v t="$title" '$1 == t { print $2; exit }' "$WORK/bag_leads.tsv")"
  [ -n "$url" ] || continue
  curl -sf -A "$UA" --retry 3 --max-filesize "$MAX_IMAGE_BYTES" -o "$DEST/$name.png.part" "$url" \
    || { rm -f "$DEST/$name.png.part"; continue; }
  new_width="$(png_width "$DEST/$name.png.part")"
  if [ ! -s "$DEST/$name.png" ]; then
    mv "$DEST/$name.png.part" "$DEST/$name.png"; filled=$((filled + 1))
  elif [ "$new_width" -gt "$(png_width "$DEST/$name.png")" ]; then
    mv "$DEST/$name.png.part" "$DEST/$name.png"; upgraded=$((upgraded + 1))
  else
    rm -f "$DEST/$name.png.part" # not larger than what is already there
  fi
done < "$WORK/todo.tsv"
echo "1. Bulbapedia bag images: $filled new, $upgraded replaced a smaller image, $skipped lead images skipped (not bag or Dream images)"

# --- 2. PokeAPI sprites for whatever is still missing -------------------
download_pokeapi() {
  local name="$1" out="$DEST/$1.png"
  [ -s "$out" ] && return 0
  curl -sf --retry 3 -o "$out.part" "$POKEAPI_URL/$name.png" && mv "$out.part" "$out" \
    || rm -f "$out.part"
}
export -f download_pokeapi
export DEST POKEAPI_URL

while IFS=$'\t' read -r name _rest; do [ -s "$DEST/$name.png" ] || echo "$name"; done < "$WORK/items.tsv" | xargs -P "$PARALLEL" -I{} bash -c 'download_pokeapi "$0"' {}
echo "2. PokeAPI fills the rest: $(count_images) images on disk"


# --- 3. Bulbapedia files by name ----------------------------------------
# The widest existing "File:Bag <Name> <game> Sprite.png" or
# "File:Dream <Name> Sprite.png" (up to MAX_WIDTH px; newest game wins a tie).
MAX_WIDTH=300
CHECKED="$HERE/seed/.item-images-checked"
touch "$CHECKED"
bulba_best_image() {
  local title="$1" titles="File:Dream $1 Sprite.png|" g
  for g in "${BAG_GAMES[@]}"; do titles+="File:Bag $title $g Sprite.png|"; done
  bulba_get --data-urlencode "action=query" --data-urlencode "prop=imageinfo" \
    --data-urlencode "iiprop=url|size" --data-urlencode "titles=${titles%|}" \
    | jq -r --arg order "${BAG_GAMES[*]}" --argjson max "$MAX_WIDTH" '
        ($order | split(" ")) as $games
        | [.query.pages[] | select(.imageinfo) | {t: .title, u: .imageinfo[0].url, w: .imageinfo[0].width}
           | select(.w <= $max)
           | . + {i: (.t as $t | [range(0; $games | length) as $k | select($t | contains(" " + $games[$k] + " Sprite")) | $k] | .[0] // 99)}]
        | sort_by([(.w * -1), .i]) | .[0].u // empty'
}

by_name=0
holdable_missing=()
while IFS=$'\t' read -r name display _category holdable; do
  if [ -s "$DEST/$name.png" ] && [ "$(png_width "$DEST/$name.png")" -gt "$SMALL_MAX_WIDTH" ]; then
    continue # already has a large image
  fi
  if ! grep -qx "$name" "$CHECKED"; then
    url="$(bulba_best_image "$(page_title "$display")" || true)"
    improved=0
    if [ -n "$url" ] && curl -sf -A "$UA" --retry 3 --max-filesize "$MAX_IMAGE_BYTES" -o "$DEST/$name.png.part" "$url"; then
      if [ ! -s "$DEST/$name.png" ] || [ "$(png_width "$DEST/$name.png.part")" -gt "$(png_width "$DEST/$name.png")" ]; then
        mv "$DEST/$name.png.part" "$DEST/$name.png"
        by_name=$((by_name + 1)); improved=1
      fi
    fi
    rm -f "$DEST/$name.png.part"
    [ "$improved" = 1 ] || echo "$name" >> "$CHECKED"
    sleep 0.5 # one polite request per item against Bulbapedia
  fi
  if [ "$holdable" = "t" ] && [ ! -s "$DEST/$name.png" ]; then holdable_missing+=("$name"); fi
done < "$WORK/items.tsv"
echo "3. Bulbapedia files by name: $by_name items got a new or larger image"

# --- 4. One generic Dynamax Crystal image for the star-named crystals ----
generic_url="$(bulba_get --data-urlencode "action=query" --data-urlencode "prop=imageinfo" \
  --data-urlencode "iiprop=url" --data-urlencode "titles=File:Bag Dynamax Crystal Sprite.png" \
  | jq -r '.query.pages[0].imageinfo[0].url // empty' || true)"
crystals=0
if [ -n "$generic_url" ] && curl -sf -A "$UA" --retry 3 -o "$WORK/dynamax-crystal.png" "$generic_url"; then
  while IFS=$'\t' read -r name _display category _holdable; do
    [ "$category" = "dynamax-crystals" ] || continue
    [ -s "$DEST/$name.png" ] && continue
    cp "$WORK/dynamax-crystal.png" "$DEST/$name.png"
    crystals=$((crystals + 1))
  done < "$WORK/items.tsv"
else
  echo "4. could not fetch the generic Dynamax Crystal image" >&2
fi
echo "4. Dynamax Crystals sharing the generic image: $crystals"

total=$(wc -l < "$WORK/items.tsv" | tr -d ' ')
have=$(count_images)
echo "Done: $have of $total items have an image; $((total - have)) will show the placeholder icon."
if [ "${#holdable_missing[@]}" -gt 0 ]; then
  echo "Holdable items without an image (${#holdable_missing[@]}): ${holdable_missing[*]}" >&2
fi
echo "Run 'docker compose up minio-init' to upload them."
