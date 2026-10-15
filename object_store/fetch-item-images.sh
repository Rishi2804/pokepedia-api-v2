#!/usr/bin/env bash
# Downloads item images into the MinIO seed folder as items/<slug>.png,
# matching the `sprite` key stored on every row of public.item.
#
# Sources, in order; each only fills items the earlier ones left without one:
#   1. The PokeAPI sprites repo, for every item (about half have one).
#   2. Bulbapedia, for every other item: the lead image of the item's page,
#      but only when it is a bag image ("Bag <Name> ... Sprite.png"). Lead
#      images that are menu screenshots or artwork are skipped, because a
#      wrong picture is worse than the placeholder icon.
#   3. Bulbapedia again, for holdable items only (the team builder's picker):
#      the newest "Bag <Name> <game> Sprite.png" that exists.
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

# Bulbapedia's bag image suffixes, newest game first.
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

# --- 1. PokeAPI ---------------------------------------------------------
download_pokeapi() {
  local name="$1" out="$DEST/$1.png"
  [ -s "$out" ] && return 0
  curl -sf --retry 3 -o "$out.part" "$POKEAPI_URL/$name.png" && mv "$out.part" "$out" \
    || rm -f "$out.part"
}
export -f download_pokeapi
export DEST POKEAPI_URL

cut -f1 "$WORK/items.tsv" | xargs -P "$PARALLEL" -I{} bash -c 'download_pokeapi "$0"' {}
echo "1. PokeAPI: $(count_images) images on disk"

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

# --- 2. Bulbapedia lead images, bag images only -------------------------
# slug<TAB>title for every item still without an image.
: > "$WORK/todo.tsv"
while IFS=$'\t' read -r name display _category _holdable; do
  [ -s "$DEST/$name.png" ] || printf '%s\t%s\n' "$name" "$(page_title "$display")" >> "$WORK/todo.tsv"
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

# Keep only bag images, then download for every slug that title serves.
grep -E $'\t.*/Bag_[^/]*_Sprite\\.png$' "$WORK/leads.tsv" > "$WORK/bag_leads.tsv" || true
skipped=$(( $(wc -l < "$WORK/leads.tsv") - $(wc -l < "$WORK/bag_leads.tsv") ))

fetched=0
while IFS=$'\t' read -r name title; do
  url="$(awk -F'\t' -v t="$title" '$1 == t { print $2; exit }' "$WORK/bag_leads.tsv")"
  [ -n "$url" ] || continue
  if curl -sf -A "$UA" --retry 3 --max-filesize "$MAX_IMAGE_BYTES" -o "$DEST/$name.png.part" "$url"; then
    mv "$DEST/$name.png.part" "$DEST/$name.png"
    fetched=$((fetched + 1))
  else
    rm -f "$DEST/$name.png.part"
  fi
done < "$WORK/todo.tsv"
echo "2. Bulbapedia bag images: $fetched fetched, $skipped lead images skipped (not bag images)"

# --- 3. Bulbapedia bag images by game, holdable items only --------------
# First existing "File:Bag <Name> <game> Sprite.png", newest game first.
bulba_bag_image() {
  local title="$1" titles="" g
  for g in "${BAG_GAMES[@]}"; do titles+="File:Bag $title $g Sprite.png|"; done
  bulba_get --data-urlencode "action=query" --data-urlencode "prop=imageinfo" \
    --data-urlencode "iiprop=url" --data-urlencode "titles=${titles%|}" \
    | jq -r --arg order "${BAG_GAMES[*]}" '
        ($order | split(" ")) as $games
        | [.query.pages[] | select(.imageinfo) | {t: .title, u: .imageinfo[0].url}]
        | map(. + {i: (.t as $t | [range(0; $games | length)] | map(select($t | contains(" " + $games[.] + " Sprite"))) | .[0] // 99)})
        | sort_by(.i) | .[0].u // empty'
}

by_game=0
holdable_missing=()
while IFS=$'\t' read -r name display _category holdable; do
  [ "$holdable" = "t" ] || continue
  [ -s "$DEST/$name.png" ] && continue
  url="$(bulba_bag_image "$(page_title "$display")" || true)"
  if [ -n "$url" ] && curl -sf -A "$UA" --retry 3 --max-filesize "$MAX_IMAGE_BYTES" -o "$DEST/$name.png.part" "$url"; then
    mv "$DEST/$name.png.part" "$DEST/$name.png"
    by_game=$((by_game + 1))
  else
    rm -f "$DEST/$name.png.part"
    holdable_missing+=("$name")
  fi
  sleep 0.5
done < "$WORK/items.tsv"
echo "3. Bulbapedia bag images by game: $by_game holdable items fetched"

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
