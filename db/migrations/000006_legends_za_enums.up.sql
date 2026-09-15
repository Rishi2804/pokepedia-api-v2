-- Legends: Z-A and its Mega Dimension DLC -- enum members only.
--
-- Split from the data load in 000007 because Postgres refuses to *use* an enum
-- label added by the same transaction ("unsafe use of new value of enum type").
-- PG 12+ permits ALTER TYPE ... ADD VALUE inside a transaction block; it does
-- not permit use. golang-migrate's postgres driver hands each file to lib/pq as
-- one argument-less Exec, which travels over the simple query protocol and runs
-- as a single implicit transaction -- so one file is one transaction whether or
-- not it writes its own BEGIN/COMMIT. Dropping the BEGIN below would NOT make a
-- single-file version legal; two files is the only split that works.
--
-- Release order is load-bearing (see 000003): Postgres sorts enums by
-- declaration order, so ORDER BY on these columns yields release order for
-- free. Every existing member has contiguous sortorder 1..N and Z-A shipped
-- after Scarlet/Violet, so plain appends are correct -- no BEFORE/AFTER.
--
-- Mega Dimension is deliberately absent from game and "group". Upstream mints
-- it as its own version group (31), but it is Z-A DLC and 000003 fixed the
-- convention: DLC is a Pokedex dimension, never a game/version-group one.
-- It enters the schema solely as the `hyperspace` dex member, exactly as
-- the-teal-mask and the-indigo-disk were folded onto scarlet/violet.
--
-- `lumiose` is upstream's `lumiose-city` shortened; the dex vocabulary is ours
-- (cf. `letsgo-kanto`, `extended-sinnoh`), unlike pokemon/move/ability ids
-- which stay verbatim PokeAPI.
--
-- Pokemon Champions (version group 32) is excluded in full, so it gets no
-- member here. Note that of the four labels below only the two `dex` ones
-- carry rows today: with Champions out there is no upstream flavour text and
-- no upstream learnset for Z-A, so game/"group" 'legends-za' are declarative.
-- They are added anyway because the enum should mirror reality rather than
-- only populated reality, and because Postgres cannot add a label to the
-- middle of an enum later -- release order would be lost.
--
-- IF NOT EXISTS keeps this replayable: the matching .down.sql cannot remove
-- enum members, so a down/up cycle would otherwise fail on a duplicate label.

BEGIN;

ALTER TYPE public.game    ADD VALUE IF NOT EXISTS 'legends-za';
ALTER TYPE public."group" ADD VALUE IF NOT EXISTS 'legends-za';

ALTER TYPE public.dex     ADD VALUE IF NOT EXISTS 'lumiose';
ALTER TYPE public.dex     ADD VALUE IF NOT EXISTS 'hyperspace';

COMMIT;
