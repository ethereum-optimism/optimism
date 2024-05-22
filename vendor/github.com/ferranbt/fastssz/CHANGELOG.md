# 0.1.2 (26 Aug, 2022)

- feat: Add `HashFn` abstraction and introduce `gohashtree` hashing [[GH-95](https://github.com/ferranbt/fastssz/issues/95)]
- feat: `sszgen` for alias to byte array [[GH-55](https://github.com/ferranbt/fastssz/issues/55)]
- feat: `sszgen` include version in generated header file [[GH-101](https://github.com/ferranbt/fastssz/issues/101)]
- feat: support `time.Time` type as native object [[GH-100](https://github.com/ferranbt/fastssz/issues/100)]
- fix: Allocate nil data structures in HTR [[GH-98](https://github.com/ferranbt/fastssz/issues/98)]
- fix: Allocate uint slice if len is 0 instead of nil [[GH-96](https://github.com/ferranbt/fastssz/issues/96)]
- feat: Simplify the logic of the merkleizer [[GH-94](https://github.com/ferranbt/fastssz/issues/94)]

# 0.1.1 (1 July, 2022)

- Struct field not as a pointer [[GH-54](https://github.com/ferranbt/fastssz/issues/54)]
- Embed container structs [[GH-86](https://github.com/ferranbt/fastssz/issues/86)]
- Introduce `GetTree` to return the tree proof of the generated object [[GH-64](https://github.com/ferranbt/fastssz/issues/64)]
- Update to go `1.8` version [[GH-80](https://github.com/ferranbt/fastssz/issues/80)]
- Fix `alias` should not be considered objects but only used as types [[GH-76](https://github.com/ferranbt/fastssz/issues/76)]
- Fix the exclude of types from generation if they are set with the `exclude-objs` flag [[GH-76](https://github.com/ferranbt/fastssz/issues/76)]
- Add `version` command to `sszgen` [[GH-74](https://github.com/ferranbt/fastssz/issues/74)]
- Support `bellatrix`, `altair` and `phase0` forks in spec tests command to `sszgen` [[GH-73](https://github.com/ferranbt/fastssz/issues/73)]

# 0.1.0 (15 May, 2022)

- Initial public release.
