package main

import (
	"fmt"
	"strings"
	"testing"
)

// The four fixtures below are real, unmodified Bulbapedia wikitext (fetched
// 2026-08-27), used to validate the gen-to-version-group mapping in
// eggmoves.go against ground truth rather than invented text.

// bulbasaurGen9BreedingFixture is Bulbasaur's main-page breeding section:
// 4 entries, no form headers, no game markers. Confirmed against this
// database's own data: Bulbasaur has exactly 4 egg moves under
// scarlet-violet.
const bulbasaurGen9BreedingFixture = `{{learnlist/breedh/9|Bulbasaur|Grass|Poison|2}}
{{learnlist/breed9|{{MSP/H|0079|Slowpoke}}{{MSP/H|0079-Galar|Slowpoke}}{{MSP/H|0080|Slowbro}}{{MSP/H|0080-Galar|Slowbro}}{{MSP/H|0199|Slowking}}{{MSP/H|0199-Galar|Slowking}}{{MSP/H|0387|Turtwig}}{{MSP/H|0388|Grotle}}{{MSP/H|0389|Torterra}}{{MSP/H|0708|Phantump}}{{MSP/H|0709|Trevenant}}{{MSP/H|0712|Bergmite}}{{MSP/H|0713|Avalugg}}{{MSP/H|0713-Hisui|Avalugg}}{{MSP/H|0842|Appletun}}{{MSP/H|0946|Bramblin}}{{MSP/H|0947|Brambleghast}}|Curse|Ghost|Status|—|—|10}}
{{learnlist/breed9|{{MSP/H|0192|Sunflora}}{{MSP/H|0331|Cacnea}}{{MSP/H|0332|Cacturne}}{{MSP/H|0459|Snover}}{{MSP/H|0460|Abomasnow}}{{MSP/H|0590|Foongus}}{{MSP/H|0591|Amoonguss}}{{MSP/H|0708|Phantump}}{{MSP/H|0709|Trevenant}}{{MSP/H|0753|Fomantis}}{{MSP/H|0754|Lurantis}}|Ingrain|Grass|Status|—|—|20}}
{{learnlist/breed9|{{MSP/H|0003|Venusaur}}{{MSP/H|0043|Oddish}}{{MSP/H|0044|Gloom}}{{MSP/H|0045|Vileplume}}{{MSP/H|0182|Bellossom}}{{MSP/H|0154|Meganium}}{{MSP/H|0192|Sunflora}}{{MSP/H|0764|Comfey}}{{MSP/H|0930|Arboliva}}|Petal Dance|Grass|Special|120|100|10||'''}}
{{learnlist/breed9|{{MSP/H|0043|Oddish}}{{MSP/H|0044|Gloom}}{{MSP/H|0045|Vileplume}}{{MSP/H|0182|Bellossom}}{{MSP/H|0199-Galar|Slowking}}{{MSP/H|0285|Shroomish}}{{MSP/H|0286|Breloom}}{{MSP/H|0590|Foongus}}{{MSP/H|0591|Amoonguss}}{{MSP/H|0757|Salandit}}|Toxic|Poison|Status|—|90|10}}
{{learnlist/breedf/9|Bulbasaur|Grass|Poison|2}}`

// bulbasaurGen4BreedingFixture is Bulbasaur (Pokémon)/Generation IV
// learnset's breeding section: 14 entries, 2 of them ("Power Whip",
// "Sludge") carrying a trailing "HGSS" token. Confirmed against this
// database: diamond-pearl=12, platinum=12, heartgold-soulsilver=14 --- the
// HGSS-tagged entries are HGSS-exclusive, not additive to the other two.
const bulbasaurGen4BreedingFixture = `{{learnlist/breedh|Bulbasaur|grass|poison|4|1}}
{{learnlist/breed4|{{MSP/3|079|Slowpoke}}{{MSP/3|080|Slowbro}}{{MSP/3|143|Snorlax}}|Amnesia|Psychic|Status|—|—|20|Cute|0}}
{{learnlist/breed4|{{MSP/3|285|Shroomish}}{{MSP/3|286|Breloom}}|Charm|Normal|Status|—|100|20|Cute|2|*}}
{{learnlist/breed4|{{MSP/3|079|Slowpoke}}{{MSP/3|080|Slowbro}}{{MSP/3|199|Slowking}}{{MSP/3|387|Turtwig}}{{MSP/3|388|Grotle}}{{MSP/3|389|Torterra}}|Curse|???|Status|—|—|10|Tough|0}}
{{learnlist/breed4|{{MSP/3|191|Sunkern}}{{MSP/3|192|Sunflora}}{{MSP/3|315|Roselia}}{{MSP/3|459|Snover}}{{MSP/3|460|Abomasnow}}|GrassWhistle|Grass|Status|—|55|15|Smart|2}}
{{learnlist/breed4|{{MSP/3|114|Tangela}}{{MSP/3|465|Tangrowth}}{{MSP/3|191|Sunkern}}{{MSP/3|192|Sunflora}}{{MSP/3|315|Roselia}}{{MSP/3|331|Cacnea}}{{MSP/3|332|Cacturne}}{{MSP/3|455|Carnivine}}{{MSP/3|459|Snover}}{{MSP/3|460|Abomasnow}}|Ingrain|Grass|Status|—|—|20|Smart|0}}
{{learnlist/breed4|{{MSP/3|071|Victreebel}}{{MSP/3|103|Exeggutor}}{{MSP/3|182|Bellossom}}{{MSP/3|192|Sunflora}}{{MSP/3|253|Grovyle}}{{MSP/3|254|Sceptile}}{{MSP/3|275|Shiftry}}{{MSP/3|357|Tropius}}{{MSP/3|387|Turtwig}}{{MSP/3|388|Grotle}}{{MSP/3|389|Torterra}}|Leaf Storm|Grass|Special|140|90|5|Cute|2||'''}}
{{learnlist/breed4|{{MSP/3|152|Chikorita}}{{MSP/3|153|Bayleef}}{{MSP/3|154|Meganium}}{{MSP/3|179|Mareep}}{{MSP/3|180|Flaaffy}}{{MSP/3|181|Ampharos}}|Light Screen|Psychic|Status|—|—|30|Beauty|2}}
{{learnlist/breed4|{{MSP/3|152|Chikorita}}{{MSP/3|153|Bayleef}}{{MSP/3|154|Meganium}}{{MSP/3|182|Bellossom}}{{MSP/3|315|Roselia}}{{MSP/3|407|Roserade}}{{MSP/3|357|Tropius}}{{MSP/3|420|Cherubi}}{{MSP/3|421|Cherrim}}|Magical Leaf|Grass|Special|60|—|20|Beauty|2||'''}}
{{learnlist/breed4|{{MSP/3|270|Lotad}}{{MSP/3|271|Lombre}}{{MSP/3|272|Ludicolo}}{{MSP/3|273|Seedot}}{{MSP/3|274|Nuzleaf}}|Nature Power|Normal|Status|—|—|20|Beauty|2}}
{{learnlist/breed4|{{MSP/3|003|Venusaur}}{{MSP/3|043|Oddish}}{{MSP/3|044|Gloom}}{{MSP/3|045|Vileplume}}{{MSP/3|154|Meganium}}{{MSP/3|192|Sunflora}}{{MSP/3|315|Roselia}}{{MSP/3|421|Cherrim}}|Petal Dance|Grass|Special|90|100|20|Beauty|0||'''}}
{{learnlist/breed4|{{MSP/3|108|Lickitung}}{{MSP/3|463|Lickilicky}}{{MSP/3|114|Tangela}}{{MSP/3|465|Tangrowth}}{{MSP/3|455|Carnivine}}|Power Whip|Grass|Physical|120|85|10|Beauty|3||'''|HGSS}}
{{learnlist/breed4|{{MSP/3|131|Lapras}}{{MSP/3|152|Chikorita}}{{MSP/3|153|Bayleef}}{{MSP/3|154|Meganium}}|Safeguard|Normal|Status|—|—|25|Beauty|2}}
{{learnlist/breed4|{{MSP/3|007|Squirtle}}{{MSP/3|008|Wartortle}}{{MSP/3|009|Blastoise}}|Skull Bash|Normal|Physical|100|100|15|Tough|1}}
{{learnlist/breed4|{{MSP/3|258|Mudkip}}{{MSP/3|259|Marshtomp}}{{MSP/3|260|Swampert}}|Sludge|Poison|Special|65|100|20|Tough|2|*|'''|HGSS}}
{{learnlist/breedf|Bulbasaur|grass|poison|4|1}}`

// bulbasaurGen8BreedingFixture is Bulbasaur (Pokémon)/Generation VIII
// learnset's breeding section: an {{gameabbrev8|SwSh}} block (6 entries)
// followed by a {{gameabbrev8|BDSP}} block (12 entries). Confirmed
// against this database: sword-shield=6, brilliant-diamond-and-shining-
// pearl=12.
const bulbasaurGen8BreedingFixture = `{{gameabbrev8|SwSh}}
{{learnlist/breedh/8|Bulbasaur|Grass|Poison|2}}
{{learnlist/breed8|{{MSP/8c|079|Slowpoke}}{{MSP/8c|079G|Slowpoke}}{{MSP/8c|080|Slowbro}}{{MSP/8c|080G|Slowbro}}{{MSP/8c|199|Slowking}}{{MSP/8c|199G|Slowking}}{{MSP/8c|597|Ferroseed}}{{MSP/8c|598|Ferrothorn}}{{MSP/8c|708|Phantump}}{{MSP/8c|709|Trevenant}}{{MSP/8c|712|Bergmite}}{{MSP/8c|713|Avalugg}}{{MSP/8c|842|Appletun}}|Curse|Ghost|Status|—|—|10}}
{{learnlist/breed8|{{MSP/8c|114|Tangela}}{{MSP/8c|465|Tangrowth}}{{MSP/8c|315|Roselia}}{{MSP/8c|407|Roserade}}{{MSP/8c|459|Snover}}{{MSP/8c|460|Abomasnow}}{{MSP/8c|556|Maractus}}{{MSP/8c|590|Foongus}}{{MSP/8c|591|Amoonguss}}{{MSP/8c|597|Ferroseed}}{{MSP/8c|598|Ferrothorn}}{{MSP/8c|708|Phantump}}{{MSP/8c|709|Trevenant}}{{MSP/8c|753|Fomantis}}{{MSP/8c|754|Lurantis}}{{MSP/8c|755|Morelull}}{{MSP/8c|756|Shiinotic}}|Ingrain|Grass|Status|—|—|20}}
{{learnlist/breed8|{{MSP/8c|270|Lotad}}{{MSP/8c|271|Lombre}}{{MSP/8c|272|Ludicolo}}{{MSP/8c|273|Seedot}}{{MSP/8c|274|Nuzleaf}}{{MSP/8c|275|Shiftry}}|Nature Power|Normal|Status|—|—|20}}
{{learnlist/breed8|{{MSP/8c|003|Venusaur}}{{MSP/8c|043|Oddish}}{{MSP/8c|044|Gloom}}{{MSP/8c|045|Vileplume}}{{MSP/8c|182|Bellossom}}{{MSP/8c|315|Roselia}}{{MSP/8c|407|Roserade}}{{MSP/8c|421|Cherrim}}{{MSP/8c|556|Maractus}}{{MSP/8c|764|Comfey}}|Petal Dance|Grass|Special|120|100|10||'''}}
{{learnlist/breed8|{{MSP/8c|007|Squirtle}}{{MSP/8c|008|Wartortle}}{{MSP/8c|009|Blastoise}}{{MSP/8c|713|Avalugg}}|Skull Bash|Normal|Physical|130|100|10}}
{{learnlist/breed8|{{MSP/8c|032|Nidoran♂}}{{MSP/8c|033|Nidorino}}{{MSP/8c|034|Nidoking}}{{MSP/8c|043|Oddish}}{{MSP/8c|044|Gloom}}{{MSP/8c|045|Vileplume}}{{MSP/8c|182|Bellossom}}{{MSP/8c|315|Roselia}}{{MSP/8c|407|Roserade}}{{MSP/8c|590|Foongus}}{{MSP/8c|591|Amoonguss}}{{MSP/8c|757|Salandit}}|Toxic|Poison|Status|—|90|10}}
{{learnlist/breedf/8|Bulbasaur|Grass|Poison|2}}

{{gameabbrev8|BDSP}}
{{learnlist/breedh/8|Bulbasaur|Grass|Poison|2}}
{{learnlist/breed8|{{MSP/H|0079|Slowpoke}}{{MSP/H|0080|Slowbro}}{{MSP/H|0199|Slowking}}{{MSP/H|0143|Snorlax}}{{MSP/H|0258|Mudkip}}{{MSP/H|0259|Marshtomp}}{{MSP/H|0260|Swampert}}|Amnesia|Psychic|Status|&mdash;|&mdash;|20||}}
{{learnlist/breed8|{{MSP/H|0285|Shroomish}}{{MSP/H|0286|Breloom}}|Charm|Fairy|Status|&mdash;|100|20|*|}}
{{learnlist/breed8|{{MSP/H|0079|Slowpoke}}{{MSP/H|0080|Slowbro}}{{MSP/H|0199|Slowking}}{{MSP/H|0387|Turtwig}}{{MSP/H|0388|Grotle}}{{MSP/H|0389|Torterra}}|Curse|Ghost|Status|&mdash;|&mdash;|10||}}
{{learnlist/breed8|{{MSP/H|0043|Oddish}}{{MSP/H|0044|Gloom}}{{MSP/H|0045|Vileplume}}{{MSP/H|0182|Bellossom}}{{MSP/H|0114|Tangela}}{{MSP/H|0465|Tangrowth}}{{MSP/H|0407|Roserade}}|Grassy Terrain|Grass|Status|&mdash;|&mdash;|10||}}
{{learnlist/breed8|{{MSP/H|0114|Tangela}}{{MSP/H|0465|Tangrowth}}{{MSP/H|0191|Sunkern}}{{MSP/H|0192|Sunflora}}{{MSP/H|0315|Roselia}}{{MSP/H|0407|Roserade}}{{MSP/H|0331|Cacnea}}{{MSP/H|0332|Cacturne}}{{MSP/H|0455|Carnivine}}{{MSP/H|0459|Snover}}{{MSP/H|0460|Abomasnow}}|Ingrain|Grass|Status|&mdash;|&mdash;|20||}}
{{learnlist/breed8|{{MSP/H|0071|Victreebel}}{{MSP/H|0103|Exeggutor}}{{MSP/H|0192|Sunflora}}{{MSP/H|0252|Treecko}}{{MSP/H|0253|Grovyle}}{{MSP/H|0254|Sceptile}}{{MSP/H|0357|Tropius}}{{MSP/H|0387|Turtwig}}{{MSP/H|0388|Grotle}}{{MSP/H|0389|Torterra}}|Leaf Storm|Grass|Special|130|90|5||'''}}
{{learnlist/breed8|{{MSP/H|0152|Chikorita}}{{MSP/H|0153|Bayleef}}{{MSP/H|0154|Meganium}}{{MSP/H|0315|Roselia}}{{MSP/H|0407|Roserade}}{{MSP/H|0357|Tropius}}{{MSP/H|0420|Cherubi}}{{MSP/H|0421|Cherrim}}|Magical Leaf|Grass|Special|60|&mdash;|20||'''}}
{{learnlist/breed8|{{MSP/H|0152|Chikorita}}{{MSP/H|0153|Bayleef}}{{MSP/H|0154|Meganium}}{{MSP/H|0270|Lotad}}{{MSP/H|0271|Lombre}}{{MSP/H|0272|Ludicolo}}{{MSP/H|0273|Seedot}}{{MSP/H|0274|Nuzleaf}}{{MSP/H|0275|Shiftry}}|Nature Power|Normal|Status|&mdash;|&mdash;|20||}}
{{learnlist/breed8|{{MSP/H|0003|Venusaur}}{{MSP/H|0043|Oddish}}{{MSP/H|0044|Gloom}}{{MSP/H|0045|Vileplume}}{{MSP/H|0182|Bellossom}}{{MSP/H|0154|Meganium}}{{MSP/H|0192|Sunflora}}{{MSP/H|0315|Roselia}}{{MSP/H|0407|Roserade}}{{MSP/H|0421|Cherrim}}|Petal Dance|Grass|Special|120|100|10||'''}}
{{learnlist/breed8|{{MSP/H|0108|Lickitung}}{{MSP/H|0463|Lickilicky}}{{MSP/H|0114|Tangela}}{{MSP/H|0465|Tangrowth}}{{MSP/H|0455|Carnivine}}|Power Whip|Grass|Physical|120|85|10||'''}}
{{learnlist/breed8|{{MSP/H|0007|Squirtle}}{{MSP/H|0008|Wartortle}}{{MSP/H|0009|Blastoise}}|Skull Bash|Normal|Physical|130|100|10||}}
{{learnlist/breed8|{{MSP/H|0258|Mudkip}}{{MSP/H|0259|Marshtomp}}{{MSP/H|0260|Swampert}}|Sludge|Poison|Special|65|100|20|*|'''}}
{{learnlist/breedf/8|Bulbasaur|Grass|Poison|2}}`

// meowthGen9BreedingFixture is Meowth (Pokémon)'s main-page breeding
// section: three form sub-headings ("Meowth", "Alolan Meowth", "Galarian
// Meowth"), each with its own entry block, some entries shared across forms
// (Spite, Covet) and some form-exclusive (Parting Shot only under Alolan;
// Curse, Double-Edge, Night Slash only under Galarian). Also exercises
// splitTopLevel against real nested-template noise: {{MSP/H|...|form=-Alola}},
// {{tt|*|...}}, {{bag/s|Mirror Herb|SV}}.
const meowthGen9BreedingFixture = `=====Meowth=====
{{learnlist/breedh/9|Meowth|Normal|Normal|2}}
{{learnlist/breed9|{{MSP/H|0056|Mankey}}{{MSP/H|0133|Eevee}}{{MSP/H|0134|Vaporeon}}{{MSP/H|0135|Jolteon}}{{MSP/H|0136|Flareon}}{{MSP/H|0196|Espeon}}{{MSP/H|0197|Umbreon}}{{MSP/H|0470|Leafeon}}{{MSP/H|0471|Glaceon}}{{MSP/H|0700|Sylveon}}{{MSP/H|0216|Teddiursa}}{{MSP/H|0217|Ursaring}}{{MSP/H|0901|Ursaluna}}{{MSP/H|0287|Slakoth}}{{MSP/H|0289|Slaking}}{{MSP/H|0677|Espurr}}{{MSP/H|0678|Meowstic}}{{MSP/H|0820|Greedent}}{{MSP/H|0915|Lechonk}}{{MSP/H|0916|Oinkologne}}{{MSP/H|0926|Fidough}}{{MSP/H|0927|Dachsbun}}|Covet|Normal|Physical|60|100|25||'''}}
{{learnlist/breed9|{{MSP/H|0206|Dunsparce}}{{MSP/H|0982|Dudunsparce}}{{MSP/H|0220|Swinub}}{{MSP/H|0221|Piloswine}}{{MSP/H|0473|Mamoswine}}{{MSP/H|0231|Phanpy}}{{MSP/H|0287|Slakoth}}{{MSP/H|0289|Slaking}}{{MSP/H|0335|Zangoose}}{{MSP/H|0613|Cubchoo}}{{MSP/H|0614|Beartic}}{{MSP/H|0775|Komala}}{{MSP/H|0974|Cetoddle}}{{MSP/H|0975|Cetitan}}|Flail|Normal|Physical|—|100|15||'''}}
{{learnlist/breed9|{{MSP/H|0234|Stantler}}{{MSP/H|0899|Wyrdeer}}|Hypnosis|Psychic|Status|—|60|20}}
{{learnlist/breed9|{{MSP/H|0133|Eevee}}{{MSP/H|0134|Vaporeon}}{{MSP/H|0135|Jolteon}}{{MSP/H|0136|Flareon}}{{MSP/H|0196|Espeon}}{{MSP/H|0197|Umbreon}}{{MSP/H|0470|Leafeon}}{{MSP/H|0471|Glaceon}}{{MSP/H|0700|Sylveon}}{{MSP/H|0190|Aipom}}{{MSP/H|0424|Ambipom}}{{MSP/H|0209|Snubbull}}{{MSP/H|0210|Granbull}}{{MSP/H|0231|Phanpy}}{{MSP/H|0417|Pachirisu}}{{MSP/H|0572|Minccino}}{{MSP/H|0573|Cinccino}}{{MSP/H|0926|Fidough}}{{MSP/H|0927|Dachsbun}}|Last Resort|Normal|Physical|140|100|5||'''}}
{{learnlist/breed9|{{MSP/H|0037|Vulpix}}{{MSP/H|0037|Vulpix|form=-Alola}}{{MSP/H|0038|Ninetales}}{{MSP/H|0038|Ninetales|form=-Alola}}{{MSP/H|0570|Zorua|form=-Hisui}}{{MSP/H|0571|Zoroark|form=-Hisui}}{{tt|*|Hisuian Zorua learned Spite via level-up prior to Version 2.0.1; it can still learn Spite via TM177}}|Spite|Ghost|Status|—|100|10}}
{{learnlist/breed9|{{MSP/H|0025|Pikachu}}{{MSP/H|0026|Raichu}}{{MSP/H|0026-Alola|Raichu}}{{MSP/H|0037|Vulpix}}{{MSP/H|0037-Alola|Vulpix}}{{MSP/H|0038|Ninetales}}{{MSP/H|0038-Alola|Ninetales}}{{MSP/H|0054|Psyduck}}{{MSP/H|0055|Golduck}}{{MSP/H|0111|Rhyhorn}}{{MSP/H|0112|Rhydon}}{{MSP/H|0464|Rhyperior}}{{MSP/H|0128|Tauros}}{{MSP/H|0128-Paldea Combat|Tauros}}{{MSP/H|0128-Paldea Blaze|Tauros}}{{MSP/H|0128-Paldea Aqua|Tauros}}{{MSP/H|0133|Eevee}}{{MSP/H|0134|Vaporeon}}{{MSP/H|0135|Jolteon}}{{MSP/H|0136|Flareon}}{{MSP/H|0196|Espeon}}{{MSP/H|0197|Umbreon}}{{MSP/H|0470|Leafeon}}{{MSP/H|0471|Glaceon}}{{MSP/H|0700|Sylveon}}{{MSP/H|0190|Aipom}}{{MSP/H|0424|Ambipom}}{{MSP/H|0194|Wooper}}{{MSP/H|0194-Paldea|Wooper}}{{MSP/H|0195|Quagsire}}{{MSP/H|0980|Clodsire}}{{MSP/H|0209|Snubbull}}{{MSP/H|0210|Granbull}}{{MSP/H|0498|Tepig}}{{MSP/H|0499|Pignite}}{{MSP/H|0500|Emboar}}{{MSP/H|0501|Oshawott}}{{MSP/H|0502|Dewott}}{{MSP/H|0503|Samurott}}{{MSP/H|0503-Hisui|Samurott}}{{MSP/H|0522|Blitzle}}{{MSP/H|0523|Zebstrika}}{{MSP/H|0653|Fennekin}}{{MSP/H|0654|Braixen}}{{MSP/H|0655|Delphox}}{{MSP/H|0672|Skiddo}}{{MSP/H|0673|Gogoat}}{{MSP/H|0702|Dedenne}}{{MSP/H|0819|Skwovet}}{{MSP/H|0820|Greedent}}{{MSP/H|0877|Morpeko}}{{MSP/H|0906|Sprigatito}}{{MSP/H|0907|Floragato}}{{MSP/H|0908|Meowscarada}}{{MSP/H|0915|Lechonk}}{{MSP/H|0916|Oinkologne}}{{MSP/H|0926|Fidough}}{{MSP/H|0927|Dachsbun}}{{MSP/H|0971|Greavard}}{{MSP/H|0972|Houndstone}}|Tail Whip|Normal|Status|—|100|30}}
{{learnlist/breedf/9|Meowth|Normal|Normal|2}}

=====Alolan Meowth=====
{{learnlist/breedh/9|Meowth|Dark|Dark|7}}
{{learnlist/breed9|{{MSP/H|0056|Mankey}}{{MSP/H|0133|Eevee}}{{MSP/H|0134|Vaporeon}}{{MSP/H|0135|Jolteon}}{{MSP/H|0136|Flareon}}{{MSP/H|0196|Espeon}}{{MSP/H|0197|Umbreon}}{{MSP/H|0470|Leafeon}}{{MSP/H|0471|Glaceon}}{{MSP/H|0700|Sylveon}}{{MSP/H|0216|Teddiursa}}{{MSP/H|0217|Ursaring}}{{MSP/H|0901|Ursaluna}}{{MSP/H|0287|Slakoth}}{{MSP/H|0289|Slaking}}{{MSP/H|0677|Espurr}}{{MSP/H|0678|Meowstic}}{{MSP/H|0820|Greedent}}{{MSP/H|0915|Lechonk}}{{MSP/H|0916|Oinkologne}}{{MSP/H|0926|Fidough}}{{MSP/H|0927|Dachsbun}}|Covet|Normal|Physical|60|100|25|grid=8}}
{{learnlist/breed9|{{MSP/H|0206|Dunsparce}}{{MSP/H|0982|Dudunsparce}}{{MSP/H|0220|Swinub}}{{MSP/H|0221|Piloswine}}{{MSP/H|0473|Mamoswine}}{{MSP/H|0231|Phanpy}}{{MSP/H|0287|Slakoth}}{{MSP/H|0289|Slaking}}{{MSP/H|0335|Zangoose}}{{MSP/H|0613|Cubchoo}}{{MSP/H|0614|Beartic}}{{MSP/H|0775|Komala}}{{MSP/H|0974|Cetoddle}}{{MSP/H|0975|Cetitan}}|Flail|Normal|Physical|—|100|15|grid=8}}
{{learnlist/breed9|{{MSP/H|0877|Morpeko}}{{MSP/H|0944|Shroodle}}{{MSP/H|0945|Grafaiai}}|Flatter|Dark|Status|—|100|15|grid=8}}
{{learnlist/breed9|{{MSP/H|0234|Stantler}}{{MSP/H|0899|Wyrdeer}}|Hypnosis|Psychic|Status|—|60|20|grid=8}}
{{learnlist/breed9|{{bag/s|Mirror Herb|SV}}{{tt|*|The Pokémon must hold a Mirror Herb to copy this move from another Pokémon.}}|Parting Shot|Dark|Status|—|100|20|grid=8}}
{{learnlist/breed9|{{MSP/H|0037|Vulpix}}{{MSP/H|0037|Vulpix|form=-Alola}}{{MSP/H|0038|Ninetales}}{{MSP/H|0038|Ninetales|form=-Alola}}{{MSP/H|0570|Zorua|form=-Hisui}}{{MSP/H|0571|Zoroark|form=-Hisui}}{{tt|*|Hisuian Zorua learned Spite via level-up prior to Version 2.0.1; it can still learn Spite via TM177}}|Spite|Ghost|Status|—|100|10|grid=8}}
{{learnlist/breedf/9|Meowth|Dark|Dark|7}}

=====Galarian Meowth=====
{{learnlist/breedh/9|Meowth|Steel|Steel|8}}
{{learnlist/breed9|{{MSP/H|0056|Mankey}}{{MSP/H|0133|Eevee}}{{MSP/H|0134|Vaporeon}}{{MSP/H|0135|Jolteon}}{{MSP/H|0136|Flareon}}{{MSP/H|0196|Espeon}}{{MSP/H|0197|Umbreon}}{{MSP/H|0470|Leafeon}}{{MSP/H|0471|Glaceon}}{{MSP/H|0700|Sylveon}}{{MSP/H|0216|Teddiursa}}{{MSP/H|0217|Ursaring}}{{MSP/H|0901|Ursaluna}}{{MSP/H|0287|Slakoth}}{{MSP/H|0289|Slaking}}{{MSP/H|0677|Espurr}}{{MSP/H|0678|Meowstic}}{{MSP/H|0820|Greedent}}{{MSP/H|0915|Lechonk}}{{MSP/H|0916|Oinkologne}}{{MSP/H|0926|Fidough}}{{MSP/H|0927|Dachsbun}}|Covet|Normal|Physical|60|100|25}}
{{learnlist/breed9|{{MSP/H|0322|Numel}}{{MSP/H|0323|Camerupt}}{{MSP/H|0324|Torkoal}}{{MSP/H|0335|Zangoose}}{{MSP/H|0570|Zorua|form=-Hisui}}{{MSP/H|0571|Zoroark|form=-Hisui}}|Curse|Ghost|Status|—|—|10}}
{{learnlist/breed9|{{MSP/H|0128|Tauros}}{{MSP/H|0128|Tauros|form=-Paldea Combat}}{{MSP/H|0133|Eevee}}{{MSP/H|0134|Vaporeon}}{{MSP/H|0135|Jolteon}}{{MSP/H|0136|Flareon}}{{MSP/H|0196|Espeon}}{{MSP/H|0197|Umbreon}}{{MSP/H|0470|Leafeon}}{{MSP/H|0471|Glaceon}}{{MSP/H|0700|Sylveon}}{{MSP/H|0155|Cyndaquil}}{{MSP/H|0156|Quilava}}{{MSP/H|0157|Typhlosion}}{{MSP/H|0157|Typhlosion|form=-Hisui}}{{MSP/H|0161|Sentret}}{{MSP/H|0162|Furret}}{{MSP/H|0206|Dunsparce}}{{MSP/H|0982|Dudunsparce}}{{MSP/H|0231|Phanpy}}{{MSP/H|0234|Stantler}}{{MSP/H|0899|Wyrdeer}}{{MSP/H|0322|Numel}}{{MSP/H|0449|Hippopotas}}{{MSP/H|0450|Hippowdon}}{{MSP/H|0585|Deerling}}{{MSP/H|0586|Sawsbuck}}{{MSP/H|0672|Skiddo}}{{MSP/H|0673|Gogoat}}{{MSP/H|0766|Passimian}}{{MSP/H|0813|Scorbunny}}{{MSP/H|0814|Raboot}}{{MSP/H|0815|Cinderace}}{{MSP/H|0915|Lechonk}}{{MSP/H|0916|Oinkologne}}{{MSP/H|0926|Fidough}}{{MSP/H|0927|Dachsbun}}{{MSP/H|0942|Maschiff}}{{MSP/H|0943|Mabosstiff}}{{MSP/H|0967|Cyclizar}}{{MSP/H|0971|Greavard}}{{MSP/H|0972|Houndstone}}{{MSP/H|0974|Cetoddle}}{{MSP/H|0975|Cetitan}}|Double-Edge|Normal|Physical|120|100|15}}
{{learnlist/breed9|{{MSP/H|0206|Dunsparce}}{{MSP/H|0982|Dudunsparce}}{{MSP/H|0220|Swinub}}{{MSP/H|0221|Piloswine}}{{MSP/H|0473|Mamoswine}}{{MSP/H|0231|Phanpy}}{{MSP/H|0287|Slakoth}}{{MSP/H|0289|Slaking}}{{MSP/H|0335|Zangoose}}{{MSP/H|0613|Cubchoo}}{{MSP/H|0614|Beartic}}{{MSP/H|0775|Komala}}{{MSP/H|0974|Cetoddle}}{{MSP/H|0975|Cetitan}}|Flail|Normal|Physical|—|100|15}}
{{learnlist/breed9|{{MSP/H|0051|Dugtrio}}{{MSP/H|0051|Dugtrio|form=-Alola}}{{MSP/H|0052|Meowth|form=-Alola}}{{MSP/H|0053|Persian|form=-Alola}}{{MSP/H|0335|Zangoose}}{{MSP/H|0434|Stunky}}{{MSP/H|0435|Skuntank}}{{MSP/H|0461|Weavile}}{{MSP/H|0571|Zoroark}}{{MSP/H|0908|Meowscarada}}|Night Slash|Dark|Physical|70|100|15}}
{{learnlist/breed9|{{MSP/H|0037|Vulpix}}{{MSP/H|0037|Vulpix|form=-Alola}}{{MSP/H|0038|Ninetales}}{{MSP/H|0038|Ninetales|form=-Alola}}{{MSP/H|0570|Zorua|form=-Hisui}}{{MSP/H|0571|Zoroark|form=-Hisui}}{{tt|*|Hisuian Zorua learned Spite via level-up prior to Version 2.0.1; it can still learn Spite via TM177}}|Spite|Ghost|Status|—|100|10}}
{{learnlist/breedf/9|Meowth|Steel|Steel|8}}`

func TestFindBreedingHeading_VariousLevels(t *testing.T) {
	// RE2 (Go's regexp engine) has no backreferences, so matching an
	// open/close heading pair at a shared but VARIABLE level can't be one
	// regex -- findBreedingHeading tries each fixed level in turn instead.
	// This is the direct regression test for that: verified live that
	// Bulbasaur's own subpages use different levels for the identical
	// heading (Generation VI subpage: four "="; Generation VII subpage: five).
	for level := 3; level <= 6; level++ {
		eq := ""
		for i := 0; i < level; i++ {
			eq += "="
		}
		wikitext := "intro text\n" + eq + "By {{pkmn|breeding}}" + eq + "\n{{learnlist/breed9|x|Tackle}}\n" + "==" + "Next==\nmore"
		gotLevel, end, found := findBreedingHeading(wikitext)
		if !found {
			t.Fatalf("level %d: heading not found", level)
		}
		if gotLevel != level {
			t.Fatalf("level %d: findBreedingHeading reported level %d", level, gotLevel)
		}
		if !strings.Contains(wikitext[end:], "learnlist/breed9") {
			t.Fatalf("level %d: end position landed past the entry", level)
		}
	}
}

func TestParseBreedingSection_Gen9_Bulbasaur(t *testing.T) {
	// bulbasaurGen9BreedingFixture is already the extracted section body
	// (see fixture comment) -- the level (4) was confirmed when it was
	// captured, so there is no heading left to re-find here.
	sec, level := bulbasaurGen9BreedingFixture, 4

	idx := buildSpeciesIndex([]pokemonRow{{ID: 1, Name: "bulbasaur", SpeciesID: 1}})
	moveIndex := map[string]int32{
		normalizeMoveKey("Curse"): 1, normalizeMoveKey("Ingrain"): 2,
		normalizeMoveKey("Petal Dance"): 3, normalizeMoveKey("Toxic"): 4,
	}

	rows, unresolved := parseBreedingSection(sec, level, genConfigs["IX"], 1, idx, "Bulbasaur (Pokémon)", moveIndex)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4: %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r.Version != "scarlet-violet" {
			t.Fatalf("row %+v: version = %q, want scarlet-violet", r, r.Version)
		}
		if r.PokemonID != 1 {
			t.Fatalf("row %+v: pokemon_id = %d, want 1", r, r.PokemonID)
		}
	}
}

func TestParseBreedingSection_Gen4_HGSSOverride(t *testing.T) {
	sec, level := bulbasaurGen4BreedingFixture, 4

	idx := buildSpeciesIndex([]pokemonRow{{ID: 1, Name: "bulbasaur", SpeciesID: 1}})
	names := []string{"Amnesia", "Charm", "Curse", "Grass Whistle", "Ingrain", "Leaf Storm",
		"Light Screen", "Magical Leaf", "Nature Power", "Petal Dance", "Power Whip",
		"Safeguard", "Skull Bash", "Sludge"}
	moveIndex := map[string]int32{}
	for i, n := range names {
		moveIndex[normalizeMoveKey(n)] = int32(i + 1)
	}

	rows, unresolved := parseBreedingSection(sec, level, genConfigs["IV"], 1, idx, "Bulbasaur (Pokémon)/Generation IV learnset", moveIndex)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none (GrassWhistle without a space must still resolve via normalizeMoveKey)", unresolved)
	}

	counts := map[string]int{}
	byVersionMove := map[string]bool{}
	for _, r := range rows {
		counts[r.Version]++
		byVersionMove[fmt.Sprintf("%s/%d", r.Version, r.MoveID)] = true
	}
	if counts["diamond-pearl"] != 12 {
		t.Fatalf("diamond-pearl = %d, want 12", counts["diamond-pearl"])
	}
	if counts["platinum"] != 12 {
		t.Fatalf("platinum = %d, want 12", counts["platinum"])
	}
	if counts["heartgold-soulsilver"] != 14 {
		t.Fatalf("heartgold-soulsilver = %d, want 14", counts["heartgold-soulsilver"])
	}

	powerWhipID := moveIndex[normalizeMoveKey("Power Whip")]
	if byVersionMove[fmt.Sprintf("diamond-pearl/%d", powerWhipID)] {
		t.Fatal("Power Whip (HGSS-tagged) leaked into diamond-pearl -- override should replace, not add to, the default groups")
	}
	if !byVersionMove[fmt.Sprintf("heartgold-soulsilver/%d", powerWhipID)] {
		t.Fatal("Power Whip missing from heartgold-soulsilver")
	}
}

func TestParseBreedingSection_Gen8_BlockMarkers(t *testing.T) {
	sec, level := bulbasaurGen8BreedingFixture, 4

	idx := buildSpeciesIndex([]pokemonRow{{ID: 1, Name: "bulbasaur", SpeciesID: 1}})
	names := []string{"Curse", "Ingrain", "Nature Power", "Petal Dance", "Skull Bash", "Toxic",
		"Amnesia", "Charm", "Grassy Terrain", "Leaf Storm", "Magical Leaf", "Power Whip", "Sludge"}
	moveIndex := map[string]int32{}
	for i, n := range names {
		moveIndex[normalizeMoveKey(n)] = int32(i + 1)
	}

	rows, unresolved := parseBreedingSection(sec, level, genConfigs["VIII"], 1, idx, "Bulbasaur (Pokémon)/Generation VIII learnset", moveIndex)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}

	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Version]++
	}
	if counts["sword-shield"] != 6 {
		t.Fatalf("sword-shield = %d, want 6", counts["sword-shield"])
	}
	if counts["brilliant-diamond-and-shining-pearl"] != 12 {
		t.Fatalf("brilliant-diamond-and-shining-pearl = %d, want 12", counts["brilliant-diamond-and-shining-pearl"])
	}
}

func TestParseBreedingSection_Meowth_MultiForm(t *testing.T) {
	sec, level := meowthGen9BreedingFixture, 4

	const (
		meowthID = 52
		alolaID  = 10107
		galarID  = 10161
	)
	idx := buildSpeciesIndex([]pokemonRow{
		{ID: meowthID, Name: "meowth", SpeciesID: meowthID},
		{ID: alolaID, Name: "meowth-alola", SpeciesID: meowthID},
		{ID: galarID, Name: "meowth-galar", SpeciesID: meowthID},
	})
	names := []string{"Covet", "Flail", "Hypnosis", "Last Resort", "Spite", "Tail Whip",
		"Flatter", "Parting Shot", "Curse", "Double-Edge", "Night Slash"}
	moveIndex := map[string]int32{}
	for i, n := range names {
		moveIndex[normalizeMoveKey(n)] = int32(i + 1)
	}

	rows, unresolved := parseBreedingSection(sec, level, genConfigs["IX"], meowthID, idx, "Meowth (Pokémon)", moveIndex)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}

	byPokemonMove := map[int32]map[int32]bool{meowthID: {}, alolaID: {}, galarID: {}}
	for _, r := range rows {
		if byPokemonMove[r.PokemonID] == nil {
			t.Fatalf("row for unexpected pokemon_id %d: %+v", r.PokemonID, r)
		}
		byPokemonMove[r.PokemonID][r.MoveID] = true
	}

	partingShot := moveIndex[normalizeMoveKey("Parting Shot")]
	curse := moveIndex[normalizeMoveKey("Curse")]
	nightSlash := moveIndex[normalizeMoveKey("Night Slash")]
	spite := moveIndex[normalizeMoveKey("Spite")]
	covet := moveIndex[normalizeMoveKey("Covet")]

	if byPokemonMove[meowthID][partingShot] {
		t.Fatal("base Meowth incorrectly got Alolan-exclusive Parting Shot")
	}
	if !byPokemonMove[alolaID][partingShot] {
		t.Fatal("Alolan Meowth missing its exclusive Parting Shot")
	}
	if byPokemonMove[meowthID][curse] || byPokemonMove[alolaID][curse] {
		t.Fatal("Curse (Galarian-exclusive) leaked onto base or Alolan Meowth")
	}
	if !byPokemonMove[galarID][curse] || !byPokemonMove[galarID][nightSlash] {
		t.Fatal("Galarian Meowth missing its exclusive Curse/Night Slash")
	}
	// Shared moves must appear on all three forms.
	for name, id := range map[string]int32{"Spite": spite, "Covet": covet} {
		for _, pid := range []int32{meowthID, alolaID, galarID} {
			if !byPokemonMove[pid][id] {
				t.Fatalf("shared move %s missing from pokemon_id %d", name, pid)
			}
		}
	}
}

func TestNormalizeMoveKey_SpacingInsensitive(t *testing.T) {
	cases := [][2]string{
		{"Grass Whistle", "GrassWhistle"},
		{"Will-O-Wisp", "Will O Wisp"},
		{"King's Shield", "Kings Shield"},
	}
	for _, c := range cases {
		if normalizeMoveKey(c[0]) != normalizeMoveKey(c[1]) {
			t.Fatalf("normalizeMoveKey(%q)=%q != normalizeMoveKey(%q)=%q",
				c[0], normalizeMoveKey(c[0]), c[1], normalizeMoveKey(c[1]))
		}
	}
}

func TestBuildMoveNameIndex_HistoricalAliases(t *testing.T) {
	// Real unresolved failures from an -old-gens run: Bulbapedia's
	// Generation II-V learnset subpages use the vintage move name (verified
	// live, e.g. Pidgey's Generation II-V subpages all say "Faint Attack"),
	// while PokeAPI's only English name for that move is the post-Gen-VI
	// rename "Feint Attack" -- so the index must resolve both spellings to
	// the same id.
	idx := map[string]int32{
		normalizeMoveKey("Feint Attack"):   185,
		normalizeMoveKey("Smelling Salts"): 265,
		normalizeMoveKey("High Jump Kick"): 136,
	}
	for alias, current := range moveNameAliases {
		if id, ok := idx[normalizeMoveKey(current)]; ok {
			idx[normalizeMoveKey(alias)] = id
		}
	}

	cases := []struct {
		vintageName string
		wantID      int32
	}{
		{"Faint Attack", 185},
		{"SmellingSalt", 265},
		{"Hi Jump Kick", 136},
	}
	for _, c := range cases {
		id, ok := idx[normalizeMoveKey(c.vintageName)]
		if !ok {
			t.Fatalf("vintage name %q did not resolve", c.vintageName)
		}
		if id != c.wantID {
			t.Fatalf("vintage name %q resolved to %d, want %d", c.vintageName, id, c.wantID)
		}
	}
}
