#!/usr/bin/env bash
# Downloads item bag sprites into the MinIO seed folder as items/<slug>.png,
# matching the `sprite` key stored on every row of public.item.
#
# Source 1: the PokeAPI sprites repo, for every item (about half of PokeAPI's
# items have one). Source 2, only for holdable items still missing one --
# those are what the team builder's picker shows -- Bulbapedia: the item
# page's lead image, else the newest "Bag <Name> <game> Sprite.png" it can
# find. Anything still missing is listed at the end; the UI shows a
# placeholder for those, so a miss is cosmetic.
#
# Needs the pokepedia-db container running with the item migrations applied
# (000013/000014); override DB_CONTAINER / DB_USER / DB_NAME if yours differ.
# Item names for Bulbapedia come from the Showdown dump the scraper embeds.
#
# Idempotent: files already on disk are skipped, so it is safe to re-run.
# After running, `docker compose up minio-init` uploads whatever is new.
set -euo pipefail

command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="$HERE/seed/items"
SHOWDOWN_ITEMS="$HERE/../cmd/scrape/data/showdown-items.json"
POKEAPI_URL="https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/items"
BULBA_API="https://bulbapedia.bulbagarden.net/w/api.php"
UA="PokePediaScrapeTool/1.0 (+https://github.com/Rishi2804/pokepedia-api-v2)"
PARALLEL="${PARALLEL:-8}"
DB_CONTAINER="${DB_CONTAINER:-pokepedia-db}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-pokepediadb}"

# Bulbapedia's bag sprite suffixes, newest game first.
BAG_GAMES=(ZA SV SwSh LA BDSP USUM SM ORAS XY B2W2 BW HGSS Pt DP E FRLG RS C GS)

mkdir -p "$DEST"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# name<TAB>showdown_id<TAB>holdable for every item.
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -At -F $'\t' \
  -c "SELECT name, coalesce(showdown_id, ''), holdable FROM item ORDER BY id" > "$WORK/items.tsv"
[ -s "$WORK/items.tsv" ] || { echo "no items in $DB_NAME -- apply the item migrations first" >&2; exit 1; }
echo "$(wc -l < "$WORK/items.tsv" | tr -d ' ') items"

download_pokeapi() {
  local name="$1" out="$DEST/$1.png"
  [ -s "$out" ] && return 0
  curl -sf --retry 3 -o "$out.part" "$POKEAPI_URL/$name.png" && mv "$out.part" "$out" \
    || rm -f "$out.part"
}
export -f download_pokeapi
export DEST POKEAPI_URL

cut -f1 "$WORK/items.tsv" | xargs -P "$PARALLEL" -I{} bash -c 'download_pokeapi "$0"' {}
echo "Sprites on disk after the PokeAPI pass: $(find "$DEST" -name "*.png" | wc -l | tr -d " ")"

bulba_get() {
  curl -sf -G -A "$UA" --retry 3 "$BULBA_API" \
    --data-urlencode "format=json" --data-urlencode "formatversion=2" "$@"
}

# Lead image of the item's Bulbapedia page, if it has one.
bulba_page_image() {
  bulba_get --data-urlencode "action=query" --data-urlencode "prop=pageimages" \
    --data-urlencode "piprop=original" --data-urlencode "redirects=1" \
    --data-urlencode "titles=$1" | jq -r '.query.pages[0].original.source // empty'
}

# First existing "File:Bag <Name> <game> Sprite.png", newest game first.
bulba_bag_sprite() {
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

fetched=0
missing=()
while IFS=$'\t' read -r name sdid holdable; do
  [ "$holdable" = "t" ] || continue
  [ -s "$DEST/$name.png" ] && continue
  title=""
  [ -n "$sdid" ] && title="$(jq -r --arg id "$sdid" '.[] | select(.id == $id) | .name' "$SHOWDOWN_ITEMS")"
  [ -n "$title" ] || { missing+=("$name"); continue; }

  url="$(bulba_page_image "$title" || true)"
  [ -n "$url" ] || url="$(bulba_bag_sprite "$title" || true)"
  if [ -n "$url" ] && curl -sf -A "$UA" --retry 3 -o "$DEST/$name.png.part" "$url"; then
    mv "$DEST/$name.png.part" "$DEST/$name.png"
    fetched=$((fetched + 1))
  else
    rm -f "$DEST/$name.png.part"
    missing+=("$name")
  fi
  sleep 0.5 # one polite request per item against Bulbapedia
done < "$WORK/items.tsv"

echo "Bulbapedia fallback: $fetched holdable items fetched"
if [ "${#missing[@]}" -gt 0 ]; then
  echo "Holdable items still without a sprite (${#missing[@]}): ${missing[*]}" >&2
fi
echo "Done. Run 'docker compose up minio-init' to upload them."
