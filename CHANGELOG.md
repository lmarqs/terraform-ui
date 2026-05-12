# [1.3.0](https://github.com/lmarqs/terraform-ui/compare/v1.2.0...v1.3.0) (2026-05-12)


### Bug Fixes

* re-enable GPG signing in release task ([ac2d8ed](https://github.com/lmarqs/terraform-ui/commit/ac2d8edb9627b65ebbf05188e9db31d0ddd27b59))
* remove @semantic-release/github plugin (goreleaser owns the release) ([e9c1bf6](https://github.com/lmarqs/terraform-ui/commit/e9c1bf63d9680cd6ec33b9e49e05d03d8d9975f1))


### Features

* remove @semantic-release/github dependency ([b6696ea](https://github.com/lmarqs/terraform-ui/commit/b6696ea9599651f3eb98049ad21154ac0d99b9c9))

# [1.2.0](https://github.com/lmarqs/terraform-ui/compare/v1.1.0...v1.2.0) (2026-05-12)


### Bug Fixes

* disable gpg signing in release task for CI compatibility ([9fad880](https://github.com/lmarqs/terraform-ui/commit/9fad88032a38652d4885641800c0a67a949c1c16))
* skip goreleaser validation in publishCmd ([aee5758](https://github.com/lmarqs/terraform-ui/commit/aee57583b8afac0496dffbf90735717497a7dfe8))


### Features

* ci/cd pipeline with cross-platform releases ([09822d7](https://github.com/lmarqs/terraform-ui/commit/09822d7dae49c1e75b875f8495bee827c2248acb))

## [1.1.1](https://github.com/lmarqs/terraform-ui/compare/v1.1.0...v1.1.1) (2026-05-12)


### Bug Fixes

* disable gpg signing in release task for CI compatibility ([9fad880](https://github.com/lmarqs/terraform-ui/commit/9fad88032a38652d4885641800c0a67a949c1c16))

# [1.1.0](https://github.com/lmarqs/terraform-ui/compare/v1.0.0...v1.1.0) (2026-05-12)


### Bug Fixes

* correct coverage parsing and set realistic threshold (70%) ([caab9c7](https://github.com/lmarqs/terraform-ui/commit/caab9c77e3bc2efd0f2a3890e8c04bdc184c9c9e))


### Features

* ci/cd pipeline with cross-platform releases ([aa75998](https://github.com/lmarqs/terraform-ui/commit/aa759984a0ce751cbcd90df00fb8486b2267dbc4))

# [1.1.0](https://github.com/lmarqs/terraform-ui/compare/v1.0.0...v1.1.0) (2026-05-12)


### Bug Fixes

* correct coverage parsing and set realistic threshold (70%) ([caab9c7](https://github.com/lmarqs/terraform-ui/commit/caab9c77e3bc2efd0f2a3890e8c04bdc184c9c9e))


### Features

* ci/cd pipeline with cross-platform releases ([aa75998](https://github.com/lmarqs/terraform-ui/commit/aa759984a0ce751cbcd90df00fb8486b2267dbc4))

# [1.0.0](https://github.com/lmarqs/terraform-ui/compare/v0.39.0...v1.0.0) (2026-05-12)


* feat!: remove legacy bash codebase and adopt plugin architecture ([13db5bd](https://github.com/lmarqs/terraform-ui/commit/13db5bd692695940c1e5fb1e87528740cd4ebe42))


### Bug Fixes

* --project flag resolves tfui.yaml path to parent directory ([a68e878](https://github.com/lmarqs/terraform-ui/commit/a68e8786f40b221832f690ab131c9e006f516000))
* add debug logging for editor lookup failures ([17bd6c2](https://github.com/lmarqs/terraform-ui/commit/17bd6c2ba2a88eca41a8bce4ab148964ba502be5))
* add debug logging to InspectSelected for diagnosing enter-key issue ([1be5164](https://github.com/lmarqs/terraform-ui/commit/1be5164322dce919e70b0386968a6dfd2c1e62f2))
* align blastradius plugin with Plugin interface ([7c8a9b8](https://github.com/lmarqs/terraform-ui/commit/7c8a9b8702bf5f73cbde16a8fa5c135124dfb707))
* align risk and phantom plugins with Plugin interface ([a8be022](https://github.com/lmarqs/terraform-ui/commit/a8be022f72cb7e24f93681da7ba78e4a980d4378))
* allow edit on branch nodes (opens module declaration) ([b991995](https://github.com/lmarqs/terraform-ui/commit/b991995e0d2e1e4bf46f3a7343a0892cae75426f))
* allow navigation and enter to work inside filter mode ([a05d933](https://github.com/lmarqs/terraform-ui/commit/a05d933132241c9bf002187e81a5d63666b9cb38))
* apply linter fixes and update tests for session context ([24f8925](https://github.com/lmarqs/terraform-ui/commit/24f892553610b95fdf6af5f6446184278921f7f1))
* auto-discover terraform dirs recursively, not just one level deep ([5479898](https://github.com/lmarqs/terraform-ui/commit/54798983caf9422dda6256030cfe7a4161c96d37))
* bounded subsequence fuzzy search (VS Code-style) ([8b5e907](https://github.com/lmarqs/terraform-ui/commit/8b5e907df94312807fda92cb1f34bfb58e648ff8))
* cache terraform state between StateList and Show calls ([f8e6ad4](https://github.com/lmarqs/terraform-ui/commit/f8e6ad41cac56a2e1c909bf8ce89fc87da01b0ea))
* calibrate tree filter threshold with tests ([bbdb44f](https://github.com/lmarqs/terraform-ui/commit/bbdb44f0fcb790e784024706e2ee85f618cd7e97))
* cap detail pan at longest line, restore subsequence fuzzy on stripped text ([81658eb](https://github.com/lmarqs/terraform-ui/commit/81658eb14bfcbd55d07af63516aa53e428e08e0e))
* consistent keybinding ergonomics across all plugins ([acf36c2](https://github.com/lmarqs/terraform-ui/commit/acf36c2ac8e5b6793388c9f4b416f58a946616a2))
* context plugin receives config for project discovery ([f6fb577](https://github.com/lmarqs/terraform-ui/commit/f6fb5770f9c2a9202460491d3379242adb4d4559))
* correct import in app_test.go after sdk extraction ([71d8dc2](https://github.com/lmarqs/terraform-ui/commit/71d8dc2db2e1676c5712acebeeefec5d6174000b))
* correct integration tests (--dir → --project, risk expectations) ([eeb40a1](https://github.com/lmarqs/terraform-ui/commit/eeb40a14fd1ea4e895abdffd6c7708b2a6ab9161))
* ctrl+w for wrap toggle (works in all modes), true fuzzy matching ([7077503](https://github.com/lmarqs/terraform-ui/commit/7077503b38f711dc9cecb262160f5a2f8bb53af5))
* ctrl+w toggles wrap in list view, w key no longer triggers it ([983036c](https://github.com/lmarqs/terraform-ui/commit/983036c46ab99bb1cd494c7947e192bba1f22262))
* detail view now uses full viewport height ([b9e20da](https://github.com/lmarqs/terraform-ui/commit/b9e20daf013c40e182bb7683062114d6a97379ad))
* detect missing TTY before starting TUI ([068a470](https://github.com/lmarqs/terraform-ui/commit/068a4700bca2dd503a8670e08038ec8f22610e9a))
* differentiate risk levels for medium-risk types and unknowns ([46ee200](https://github.com/lmarqs/terraform-ui/commit/46ee20096001126151027fcb516405d35c42e016))
* direct 'e' key opens all pinned resources when multiple pinned ([6a7114b](https://github.com/lmarqs/terraform-ui/commit/6a7114b041f5beca8064c282106f4fb97d24baba))
* don't auto-run plan on startup, use Activatable interface ([ad6c529](https://github.com/lmarqs/terraform-ui/commit/ad6c529ee7ca69eb0a1be91e3601574d398776e1))
* enable horizontal pan in filter mode ([c8fee04](https://github.com/lmarqs/terraform-ui/commit/c8fee041584101f7eeef236636c07feeec4eb305))
* exclude internal/terraform from coverage (I/O-bound, 90% threshold) ([8dab3b7](https://github.com/lmarqs/terraform-ui/commit/8dab3b73c5a556964f247424a7a9e1c655945731))
* filter matches leaf names (not full paths), enter toggles in filter ([beb3c86](https://github.com/lmarqs/terraform-ui/commit/beb3c8614d3fabf0c445ca1b65e44cfa319bfc1d))
* flat list trailing newline consuming viewport row ([3448816](https://github.com/lmarqs/terraform-ui/commit/34488164698fee0d83db3c83001b60444c46d73d))
* flat mode selection mismatch in state plugin ([e29af89](https://github.com/lmarqs/terraform-ui/commit/e29af89920e1da8301d583452e941f70f3b9aee1))
* handle backspace across terminals (ctrl+h, delete) ([754da39](https://github.com/lmarqs/terraform-ui/commit/754da390bfe6a3f162012d747267f98a8d7c8ff2))
* handle bracket-quoted modules and indexed resources in source lookup ([a1c98b7](https://github.com/lmarqs/terraform-ui/commit/a1c98b74ab648a476d681904dce99fc0bb039bb4))
* handle EDITOR with args (e.g., "code --wait") ([8739a31](https://github.com/lmarqs/terraform-ui/commit/8739a31b2a527bcab40b17ad5faa5091312c3319))
* ignore / key inside filter mode to prevent literal slash in filter ([73b8a52](https://github.com/lmarqs/terraform-ui/commit/73b8a5214c8e393ddab49ffe7cbc4ca89b76c6ac))
* implement actual terraform workspace switching, creation, and deletion ([14cbfc7](https://github.com/lmarqs/terraform-ui/commit/14cbfc7b9371208c6f13ac49644cdd265f91fb8e))
* improve tree filter and keybindings ([2a7d569](https://github.com/lmarqs/terraform-ui/commit/2a7d569aa0a24a4c7f8a7d15ed366053a11739aa))
* increase tree filter score threshold ([efc4d5b](https://github.com/lmarqs/terraform-ui/commit/efc4d5bb42b0c507ecdce9e108f4b37329a22a8e))
* invert ^t hint label and add taint/untaint to hint bar ([c166114](https://github.com/lmarqs/terraform-ui/commit/c16611432ad3883170e728de27ce6e8d6514a3e0))
* lint fixes in plugins (formatting, unused vars) ([8617d61](https://github.com/lmarqs/terraform-ui/commit/8617d6154aa8f0ac2d8b1c6620c1693f2ec45ac1))
* list pan, wrap truncation, and fuzzy search precision in state plugin ([7204b91](https://github.com/lmarqs/terraform-ui/commit/7204b9169503f893020e60b19e8444814f176189))
* lock ID regex matched RequestID instead of actual lock ID ([35838e5](https://github.com/lmarqs/terraform-ui/commit/35838e5a92e3f58d59b8f7f5d67aa6d8dbd7744a))
* lowercase input text for fzf (case-insensitive matching) ([75fef7f](https://github.com/lmarqs/terraform-ui/commit/75fef7fee808bb07e93cd234c39477af81539e93))
* make filter case insensitive ([6fb9a7e](https://github.com/lmarqs/terraform-ui/commit/6fb9a7eacf1ea6fac82f4ef4c1dd34a155eea009))
* make FuzzyFilter case-insensitive for both text and pattern ([1bbdd9b](https://github.com/lmarqs/terraform-ui/commit/1bbdd9bfed341389b6214e0fa50003ffee0f5667))
* make input handler mode-aware for text vs confirm prompts ([68402b0](https://github.com/lmarqs/terraform-ui/commit/68402b02a94b66ca850a33a53dd821ae08905f2a))
* move help to end of status bar ([7504b16](https://github.com/lmarqs/terraform-ui/commit/7504b167f1828081e1cd2fc56c48f1d5b4239016))
* move pan next to navigate in status bar ([39ac0a8](https://github.com/lmarqs/terraform-ui/commit/39ac0a851af465e8226394959262b9200f7ed488))
* move scope filter bar to top for consistency with state plugin ([23bc239](https://github.com/lmarqs/terraform-ui/commit/23bc239dce7d41c4929e42a9c379aba6682cc320))
* only arrow keys navigate in filter mode, j/k go to filter input ([9dc30cc](https://github.com/lmarqs/terraform-ui/commit/9dc30cc3398ce53fefc70ec31d11532e250ba4b7))
* open context picker on first render, not Init ([8125250](https://github.com/lmarqs/terraform-ui/commit/81252500e24763216a130de44e8ab0d34359ac0d))
* plugins use Activatable pattern — no auto-load on startup ([19568f1](https://github.com/lmarqs/terraform-ui/commit/19568f139a4d92cb0a34f6c4595408e56dfb45b3))
* prevent multiple concurrent inspect calls, show loading state ([da3b36a](https://github.com/lmarqs/terraform-ui/commit/da3b36a32879222ee905a96feaefe4068fd3e0a3))
* propagate errors from config.Load instead of silently swallowing ([b01c204](https://github.com/lmarqs/terraform-ui/commit/b01c2045b157fa718266e336335c19ff74ab1d9d))
* reduce tree indentation, show leaf labels only ([a21ef5e](https://github.com/lmarqs/terraform-ui/commit/a21ef5e9261a6ce97011ce735d3cd07610faac13))
* relax score threshold to only drop negative scores ([9b9ded3](https://github.com/lmarqs/terraform-ui/commit/9b9ded38b99cd5f62677973275acfcb436eb03f6))
* remove = alias for + depth key ([a337908](https://github.com/lmarqs/terraform-ui/commit/a33790861a738a0ae0d1871c9f4a52c94708e58f))
* remove bare 'w' wrap toggle from detail frame ([0061c72](https://github.com/lmarqs/terraform-ui/commit/0061c72e98c98bf94dd68dce983885035a1ba0db))
* remove orphaned git submodule entries ([d5318b7](https://github.com/lmarqs/terraform-ui/commit/d5318b7e834df1916a4b2fab53269d5ebb79dd33))
* replace -t flag with graceful non-TTY degradation ([aec723f](https://github.com/lmarqs/terraform-ui/commit/aec723f642edb89715da030f4165a98cf062c5e9))
* replace subsequence with space-separated AND terms for filtering ([849061d](https://github.com/lmarqs/terraform-ui/commit/849061d75d1349bbf266143f5c18277bb8d18034))
* require explicit y/n for confirmations, not enter ([78dbe5e](https://github.com/lmarqs/terraform-ui/commit/78dbe5e2e49db8c7d4bdf3afc97b1fb3c13d81f3))
* resolve all golangci-lint violations and add version resolution ([2847e4a](https://github.com/lmarqs/terraform-ui/commit/2847e4a94cb88aa4b2bae613eeb5c87e35be70bc))
* resolve module-prefixed addresses in source index lookup ([5e0d72c](https://github.com/lmarqs/terraform-ui/commit/5e0d72c3781238112cf786a2089f8351572ef01b))
* resolve project dir to absolute path and show only basename in header ([f42f4e9](https://github.com/lmarqs/terraform-ui/commit/f42f4e9f189cb0c88c7a3406f25452395c66948a))
* respect brackets in terraform address splitting ([a25ed81](https://github.com/lmarqs/terraform-ui/commit/a25ed8130edf6b989e91e1ab8b52faea67530613))
* restore context overlay on Init (was stale build) ([1c0b113](https://github.com/lmarqs/terraform-ui/commit/1c0b113e4e2b513e221d12a1728a514012bba31a))
* restore context overlay on startup, hide from home menu ([ca9601b](https://github.com/lmarqs/terraform-ui/commit/ca9601be8c15f179043bca2ce36f7a00e46da3fa))
* restore original fzf fuzzy filter on full address ([31b965d](https://github.com/lmarqs/terraform-ui/commit/31b965da8d820840ddac3b48610c24a7e607d125))
* restore semantic-release with proper package-lock.json and npm ci ([d16c8cd](https://github.com/lmarqs/terraform-ui/commit/d16c8cd0dbdf3acb2ca3828c196fb6485250a433))
* run fzf matching against short search text, not full address ([ca7eb15](https://github.com/lmarqs/terraform-ui/commit/ca7eb15fe1dfd5c1be7a00c133cb1a64d7753348))
* security hardening — redact sensitive state, restrict log perms, cleanup plan file ([4cd2ff2](https://github.com/lmarqs/terraform-ui/commit/4cd2ff2b3c9a9da09edf7d55be64c935886ac068))
* segment-skip with min 3 chars per chunk, recursive for 3+ segments ([5cf420c](https://github.com/lmarqs/terraform-ui/commit/5cf420cce89bebbcb58f7d902b38c7dff07de27e))
* selected row respects wrap toggle (MaxWidth when off) ([23720d0](https://github.com/lmarqs/terraform-ui/commit/23720d01ba8abdb79b710c7ec75f1fa142c75417))
* show "space=AND" hint in filter mode ([f76d5ce](https://github.com/lmarqs/terraform-ui/commit/f76d5cedb968d50e1d3e3e1ec99d41a49a39e825))
* show taint/untaint hints in detail frame ([0ec5715](https://github.com/lmarqs/terraform-ui/commit/0ec5715e4c70aa978a991de7d579685571d9e688))
* show taint/untaint keys in list view hint bar ([238bae3](https://github.com/lmarqs/terraform-ui/commit/238bae3d30d2998beda0ba94a34473a0844d9d89))
* skip scope picker in read-only mode and when --scope is set ([6c781c2](https://github.com/lmarqs/terraform-ui/commit/6c781c269cd788689f00583555282f8bbb7ba6a1))
* split filter at digit/letter boundaries for index matching ([bcbe71d](https://github.com/lmarqs/terraform-ui/commit/bcbe71da3a3025b0f9ca2dbaf0d4182433475261))
* state plugin height calculation shows all available rows ([1b27da4](https://github.com/lmarqs/terraform-ui/commit/1b27da4903dcbbabe8c510f9af601a41c95ead27))
* strip common address prefix before fuzzy matching ([6c9f85c](https://github.com/lmarqs/terraform-ui/commit/6c9f85c26d6cc57ce1ada4acd652d4448f612642))
* strip separators from both input and address consistently ([5b62ac1](https://github.com/lmarqs/terraform-ui/commit/5b62ac119663389c3366b1277fc8f1815fca4d8f))
* strip shared root module from search text, add score threshold ([8b0b3fb](https://github.com/lmarqs/terraform-ui/commit/8b0b3fb7be9e85881be3cac210681d07a6b3345d))
* swap +/- depth keys and cap at max nesting ([5da5904](https://github.com/lmarqs/terraform-ui/commit/5da5904e12f4ef3c8ef4c207aae2dd8616d55b41))
* swap wrap semantics (on=fit terminal, off=unlimited width+pan) ([d4a1497](https://github.com/lmarqs/terraform-ui/commit/d4a1497a7fec284cfef80ce31a7f3ba507f1a3b0))
* tree filter monotonicity — longer queries no longer increase results ([a3cb6e4](https://github.com/lmarqs/terraform-ui/commit/a3cb6e4027b126829653b0f7322e84f1a5d89266))
* tree filter skips short queries (< 3 chars) ([cbf20b1](https://github.com/lmarqs/terraform-ui/commit/cbf20b1494962b33ceca3452e87872b6d9573f04))
* tree mode uses fzf matching without reordering ([04581e6](https://github.com/lmarqs/terraform-ui/commit/04581e6e33e18a0d25fc08d59851b182e99bff92))
* tree viewport scrolls only when cursor hits edge ([877eb4f](https://github.com/lmarqs/terraform-ui/commit/877eb4fb6867c829953772ddd2ffba68510f2b50))
* truncate long resource lines to prevent viewport overflow ([1822266](https://github.com/lmarqs/terraform-ui/commit/18222664f9ad835dcef4ec35d32b13acf6029ace))
* tune tree filter threshold to len*17 ([741b706](https://github.com/lmarqs/terraform-ui/commit/741b70616554ab98a1932ee5edd434a96ae4d001))
* update app_test.go for sdk import changes ([10ca312](https://github.com/lmarqs/terraform-ui/commit/10ca3128b4dc8050105c9717935b287b5cffbbe7))
* use [ ] for depth control instead of +/- ([b1a419d](https://github.com/lmarqs/terraform-ui/commit/b1a419d7ce0bdc323d097d5e17ad153947a54ba7))
* use awk instead of bc for coverage threshold comparison in CI ([f1659d5](https://github.com/lmarqs/terraform-ui/commit/f1659d5a98675755d86bdd2d699d5e44c260324d))
* use ctrl+t for tree toggle, tighter tree filter scoring ([8db16be](https://github.com/lmarqs/terraform-ui/commit/8db16be16173a252bce8e74927b9a2bf6fa22e39))
* use short search text (module+type+name) for fuzzy, no score cutoff ([b473e05](https://github.com/lmarqs/terraform-ui/commit/b473e0510436961e418ef6207e323d599b5fcf16))
* wrap mode respects viewport height, shows resource count ([d5a8238](https://github.com/lmarqs/terraform-ui/commit/d5a8238bfb0b2b2d3c997d31935804bfe7915260))
* wrap/pan semantics and fuzzy search algorithm ([37cb9ba](https://github.com/lmarqs/terraform-ui/commit/37cb9ba7a7b9364444c094e1e38c22ec6c18e869))


### Features

* add --config flag and configurable log directory ([e3a4928](https://github.com/lmarqs/terraform-ui/commit/e3a4928b2919054abbfc5b94da23ae694c2ee8c9))
* add --macro CLI flag for tape execution ([b2136cc](https://github.com/lmarqs/terraform-ui/commit/b2136cc9c23d71a366802235762d97b24467c05b))
* add --plan and --state CLI flags for read-only mode ([1facaaa](https://github.com/lmarqs/terraform-ui/commit/1facaaa439fcc36feef11d994a1799b0449d98c8))
* add --scope flag and support raw tfstate format ([3f45cb9](https://github.com/lmarqs/terraform-ui/commit/3f45cb9bc31f5fe53ce7b5eddcbbf60f3f348396))
* add -t/--tty flag to force TTY mode ([c090d87](https://github.com/lmarqs/terraform-ui/commit/c090d87277fbc11d22c1cfcbbaa9583cbfcb60de))
* add ^w wrap and ←→ pan to global status bar ([ab30d2a](https://github.com/lmarqs/terraform-ui/commit/ab30d2ab3e5445413f5b03f19eb93152c38e0f90))
* add AI overlay panel component with streaming support ([e9dddf8](https://github.com/lmarqs/terraform-ui/commit/e9dddf8419ef0131b01162abc912822d6b2498a3))
* add Anthropic Claude AI provider with streaming ([bf03d4e](https://github.com/lmarqs/terraform-ui/commit/bf03d4e5c1a94ab9b40f5ff9e3bab3dfc1912173))
* add app-level command mode (:) for switching views like k9s ([04e2f67](https://github.com/lmarqs/terraform-ui/commit/04e2f67f075e88dba1c2d48735b45167f8da91d1))
* add arrow key scrolling in resource detail view ([cf302d3](https://github.com/lmarqs/terraform-ui/commit/cf302d3f32230c52101bf97b211b6f7ad3c82f5d))
* add basedir config property (like TypeScript's rootDir) ([091530d](https://github.com/lmarqs/terraform-ui/commit/091530da6e163d8397fd81ec998211a48654477e))
* add blue separator lines and dynamic shortcuts in status bar ([a766ec5](https://github.com/lmarqs/terraform-ui/commit/a766ec5a74866bf488046044961beae84ecce0c4))
* add bubbletea TUI with action-oriented home screen ([6afb9a9](https://github.com/lmarqs/terraform-ui/commit/6afb9a995ce33b78ededa1f4e01e6ca338065c96))
* add checkbox-style pin indicators with partial state ([798a35e](https://github.com/lmarqs/terraform-ui/commit/798a35efca47c3fbeccdc4fc960df8f6ed53f024))
* add Claude Code agents for test writing, conventions, architecture, and security ([518ec00](https://github.com/lmarqs/terraform-ui/commit/518ec0051dff840f94778a540757c89a0b88dfb4))
* add CLI subcommands for plugin actions ([c44b4fc](https://github.com/lmarqs/terraform-ui/commit/c44b4fcf62e0086bf00ac624281669a921e51a2b))
* add collapsible depth grouping to state browser ([b700ccb](https://github.com/lmarqs/terraform-ui/commit/b700ccb034548839bbb305d14eccdc9d22f4d261))
* add command mode (:) and state caching for instant inspect ([0d3ad2d](https://github.com/lmarqs/terraform-ui/commit/0d3ad2d2cd5933b43a624d81323deefd1381aa20))
* add Command type for terraform CLI serialization ([a49c443](https://github.com/lmarqs/terraform-ui/commit/a49c443e44701e810618d193fa1e37619a8d251a))
* add CommandBar component for bordered command input ([7007baf](https://github.com/lmarqs/terraform-ui/commit/7007bafce80f428e2a10a082efddb3e4352a21f4))
* add ContentBorder component with embedded title ([83cf56b](https://github.com/lmarqs/terraform-ui/commit/83cf56be3b304a3c76d22c505ba4f1231a2b1e8e))
* add Countable interface for plugin item counts ([a080440](https://github.com/lmarqs/terraform-ui/commit/a080440a9935a5b765e4e95859090b2efbbe2c61))
* add Cursor and ExpandSet primitives to SDK ([69ced7c](https://github.com/lmarqs/terraform-ui/commit/69ced7c222f64900f1f0bebf51d1afc9542c195e))
* add editor integration and source location index ([463fcd8](https://github.com/lmarqs/terraform-ui/commit/463fcd805fb20f52cf28a2ffb6096e50cbdeff02))
* add explicit filter mode to state browser (/ to activate) ([282f2d6](https://github.com/lmarqs/terraform-ui/commit/282f2d6a220ee68783054f24de3881c1f30b332c))
* add extension interface and registry ([29c6778](https://github.com/lmarqs/terraform-ui/commit/29c6778134af32c6c57e9ed826870385c8f6ff57))
* add filter to scope picker and fix dim list text ([3ee6e03](https://github.com/lmarqs/terraform-ui/commit/3ee6e03d77a8d5470ba2a4491c7bd43879ed729a))
* add generic FuzzyFilter[T] to SDK ([f8deb0e](https://github.com/lmarqs/terraform-ui/commit/f8deb0ee8137eede07f00927fe814e4b98cae4d2))
* add go:fmt task and wire lint into build pipeline ([e5f6cde](https://github.com/lmarqs/terraform-ui/commit/e5f6cde84d88670da1d5d60a4bebc73edeae3ba1))
* add Hintable interface for state-aware status bar hints ([9b69c76](https://github.com/lmarqs/terraform-ui/commit/9b69c763fb902f694481573b5269ff707117c6c2))
* add HintSet bitmask system for consistent footer ordering ([c020a57](https://github.com/lmarqs/terraform-ui/commit/c020a57a443989d196edd9138794f3f356b3cc4b))
* add horizontal pan and wrap toggle to state list view ([a1fbfe7](https://github.com/lmarqs/terraform-ui/commit/a1fbfe74d245afa2ab1bca21f719e526f47840a2))
* add horizontal scroll (←→) in detail view, fix content height ([bdadd7c](https://github.com/lmarqs/terraform-ui/commit/bdadd7c4981b641335d3714bc06399e36d3367dd))
* add init wizard plugin for tfui.yaml generation ([8bd4876](https://github.com/lmarqs/terraform-ui/commit/8bd4876d0c34d2cdc458842a93aba90b0bfc6fba))
* add macro engine with programmatic driver and tape DSL ([ecc2bfe](https://github.com/lmarqs/terraform-ui/commit/ecc2bfec65a565f68ad21f550b7668bf4c22de36))
* add macro Runner for tape execution ([fee111f](https://github.com/lmarqs/terraform-ui/commit/fee111f491493e755bbc36d3e10f1724cdb3f993))
* add macro-runner agent for automated UI verification ([a2abd93](https://github.com/lmarqs/terraform-ui/commit/a2abd93ccf114bc9896e6147d7d28a445246056b))
* add modal overlay system with context picker ([a92e1e4](https://github.com/lmarqs/terraform-ui/commit/a92e1e419d2046e7c70a4804bc5a8b86c41a37ca))
* add move, taint, untaint, import actions to state plugin ([c23e934](https://github.com/lmarqs/terraform-ui/commit/c23e934a9cd005f324729c7e9461760f04e6b49b))
* add navigation stack SDK types (Frame, Stack, reusable frames) ([9d5139e](https://github.com/lmarqs/terraform-ui/commit/9d5139e758efed0f40b70fe5e5e5dcc3c818537b))
* add OpenTofu support with auto-detection ([ced4e9f](https://github.com/lmarqs/terraform-ui/commit/ced4e9f90c85e84d278b30dd6f61abbf5baa8a2e))
* add output plugin for viewing terraform outputs ([d799716](https://github.com/lmarqs/terraform-ui/commit/d7997167d41ac7a4aa1b2537c594142bd2515e6c))
* add pin and apply actions to plan plugin ([a3fc7b9](https://github.com/lmarqs/terraform-ui/commit/a3fc7b9d083262d4d2653b9bc567fd793a955148))
* add PinService to SDK for shared pin operations ([f2c55a8](https://github.com/lmarqs/terraform-ui/commit/f2c55a8d6ab924762bedb3fb85459c2f537893bc))
* add REPL plugin for terraform console ([ff30fd0](https://github.com/lmarqs/terraform-ui/commit/ff30fd0fe291e42402b82a5a33fa56a2c5c3519b))
* add reusable ActionFrame to SDK frames ([ebb2314](https://github.com/lmarqs/terraform-ui/commit/ebb2314f41efc60840d6d5201b455231db66a7fa))
* add reusable KeyHint constants to SDK ([fae3496](https://github.com/lmarqs/terraform-ui/commit/fae3496a6f1df040aae76b45ad5895fdf827e2d1))
* add reusable tree navigation component (pkg/sdk/ui/tree) ([61e5abb](https://github.com/lmarqs/terraform-ui/commit/61e5abb69b09e946e56a11c561a891151325cba8))
* add roadmap as Jekyll collection, remove PLAN.md ([9436c18](https://github.com/lmarqs/terraform-ui/commit/9436c1830893bf35a2662ddd0fde20bd8fbc1511))
* add ScopeGuard to SDK for scope-change detection ([2acf4ca](https://github.com/lmarqs/terraform-ui/commit/2acf4cad31db695514d6e7ec1fd085daf2f30901))
* add SDK UI primitives (viewport, input, actions, staleness, AI interface) ([90b1b8d](https://github.com/lmarqs/terraform-ui/commit/90b1b8d66c0577384fd17f093cba07e023f95ceb))
* add session cache for inter-plugin data sharing ([19d8b0b](https://github.com/lmarqs/terraform-ui/commit/19d8b0b213dde8153bbd930c4892f21d44cf06ae))
* add shared Status enum to SDK ([a9147ac](https://github.com/lmarqs/terraform-ui/commit/a9147ac0fed6428be71aa2c547335933518172a9))
* add StaticService for read-only plan/state viewing ([2f4508f](https://github.com/lmarqs/terraform-ui/commit/2f4508f5194f97e0dcb11f5183965684c0710622))
* add structured AppContext with config, cache, and terraform state ([97fe9fe](https://github.com/lmarqs/terraform-ui/commit/97fe9feb686d41fc44885b2464edf0a1a35913f8))
* add structured debug logging with slog ([c9c062b](https://github.com/lmarqs/terraform-ui/commit/c9c062b6296b5aac940313f804ef5a0945d9a68d))
* add t key to toggle between flat and tree view ([0522bca](https://github.com/lmarqs/terraform-ui/commit/0522bcaf4463f09f9ea82d979d760c24894b35e9))
* add tab autocomplete and match hints in command mode ([4349389](https://github.com/lmarqs/terraform-ui/commit/434938992827892915717c4b20b97efc33ba61db))
* add terraform service layer with risk, phantom, and grouping ([09c747c](https://github.com/lmarqs/terraform-ui/commit/09c747cebb56b1cd01f829f30c0769a6f5b8358e))
* add universal source abstraction for URI-based data loading ([8789edd](https://github.com/lmarqs/terraform-ui/commit/8789edd1580e174cb8aef94fdde7214115be36d3))
* add UX review slash command and automated UX hooks ([feac3c9](https://github.com/lmarqs/terraform-ui/commit/feac3c972e26be9d8d399671be056949cd40d855))
* add validate plugin for terraform configuration checks ([df7f033](https://github.com/lmarqs/terraform-ui/commit/df7f033ded8b8f59e94c874c21594b7e51f6c63f))
* add WithPreserveOrder option to tree package ([67a7858](https://github.com/lmarqs/terraform-ui/commit/67a785891daf5351f9e6f42867d9ea7b14bd1d9a))
* add workspace management operations to Service interface ([5f3a10e](https://github.com/lmarqs/terraform-ui/commit/5f3a10e72cd55398f40e6fb88d1895b3bb806eef))
* add wrap toggle (w key) in resource detail view ([75d66f9](https://github.com/lmarqs/terraform-ui/commit/75d66f9382b62188525313832005ee135a9aba3d))
* auto-detect AI provider from environment credentials ([5d971e5](https://github.com/lmarqs/terraform-ui/commit/5d971e5b754c4dfb0f6c9256980c3dc66b749c16))
* auto-focus filter on state entry, enter goes directly to inspect ([51e877c](https://github.com/lmarqs/terraform-ui/commit/51e877caa7985e96339ad2c3bafdd28a710ef1f8))
* blast radius analysis for destructive plan changes ([#10](https://github.com/lmarqs/terraform-ui/issues/10)) ([af3585f](https://github.com/lmarqs/terraform-ui/commit/af3585f929e83fec4df10eefec059f27bf8d6257))
* connect plan→apply transition with pinned resource targeting ([4e22065](https://github.com/lmarqs/terraform-ui/commit/4e22065eee9a7bbb00f7807b9feb36f7f5f84c6f))
* ctrl+p filter pinned only, ctrl+u clear all pins ([3bf80b3](https://github.com/lmarqs/terraform-ui/commit/3bf80b3b715de6c695f1635a733681dc38a09cf7))
* detect state locks and offer force-unlock from error view ([9412076](https://github.com/lmarqs/terraform-ui/commit/9412076f7a48244497c9122c7876d12f594c9a86))
* expand Service interface with state mutations and terraform ops ([9cb9943](https://github.com/lmarqs/terraform-ui/commit/9cb9943f4664a6a2cae8f7bf84c425b759d551bd))
* extract pkg/sdk as public plugin contract ([8caad72](https://github.com/lmarqs/terraform-ui/commit/8caad72ca9ea95ffb7e72982e0e86124156d22db))
* fuzzy multi-term filter in state browser ([23bfdec](https://github.com/lmarqs/terraform-ui/commit/23bfdece5cee039f5fc82ea17b3f204bdbfe2e25))
* implement Countable in blast radius plugin ([d487369](https://github.com/lmarqs/terraform-ui/commit/d4873697251bb01ca1b97c3491fca388f32ca1e2))
* implement Countable in output plugin ([a62a71c](https://github.com/lmarqs/terraform-ui/commit/a62a71c93b41535defe7008aecbc31d9654c6b51))
* implement Countable in plan plugin ([7f67180](https://github.com/lmarqs/terraform-ui/commit/7f671804f775b2bffc8b18b5f09842517b243140))
* implement Countable in state plugin ([1a447ae](https://github.com/lmarqs/terraform-ui/commit/1a447ae7f5a0b3cffd809b41ec0fea352bf3ea38))
* implement non-interactive CLI modes and functional TUI views ([7a98169](https://github.com/lmarqs/terraform-ui/commit/7a9816906e03fb2dcad6cbaacfe8f97bb63a34a0))
* init plugin shows menu with edit and re-init options ([343cf92](https://github.com/lmarqs/terraform-ui/commit/343cf929769ecfdac44a99f8a59c19e0fa7f52b1))
* initialize Go project with cobra CLI and config ([8cf8c2c](https://github.com/lmarqs/terraform-ui/commit/8cf8c2cc0532ec4640b9fa6ce2ade88f5e4163c3))
* integrate sahilm/fuzzy for VS Code-style search with ranking ([2976b2f](https://github.com/lmarqs/terraform-ui/commit/2976b2feeda30cc5699e1e8427365777b24d53cc))
* make wrap toggle (w) global across list and detail views ([7fed0d9](https://github.com/lmarqs/terraform-ui/commit/7fed0d9b7b8367a04e6d0419cc543879afdef037))
* match ignoring separators (no spaces needed) ([fa8f705](https://github.com/lmarqs/terraform-ui/commit/fa8f7054b3e8a5cffb74bd04ac944af5921a3229))
* modular extension system with per-extension config ([c98ec85](https://github.com/lmarqs/terraform-ui/commit/c98ec8511992cf6327d28eb00861f381618e1d39))
* open context picker on startup, restore C shortcut ([f81653a](https://github.com/lmarqs/terraform-ui/commit/f81653ae85dd97331f9fbf06a8f5c1485f1b66b9))
* open multiple pinned resources in editor simultaneously ([94cad22](https://github.com/lmarqs/terraform-ui/commit/94cad2233326bf4cc39cae7ff2db11794bd237fa))
* pinned items float to top, space pins from filter mode ([143dc86](https://github.com/lmarqs/terraform-ui/commit/143dc863b23383f793b177cf25227590d74bcbb1))
* redesign header to 3-line layout with ASCII logo ([746bfde](https://github.com/lmarqs/terraform-ui/commit/746bfdecc24075d75fb18b655386f6185cee3e79))
* redesign header with compact/expanded modes and rich state ([e920d3f](https://github.com/lmarqs/terraform-ui/commit/e920d3fe30693d52778b9b344c820b2bb5265f78))
* register init plugin and add tfui init subcommand ([1d95cc7](https://github.com/lmarqs/terraform-ui/commit/1d95cc7b7a964f9a49052af5aace70a5270b53c3))
* register new plugins and add state context actions ([d69e8c9](https://github.com/lmarqs/terraform-ui/commit/d69e8c9317e3c505e96ddd63d835733db112677c))
* replace custom fuzzy with fzf algorithm (Smith-Waterman scoring) ([237345c](https://github.com/lmarqs/terraform-ui/commit/237345caa9c54db69b81a36156e64a28c11b4cf8))
* segment-skip fuzzy matching with comprehensive test cases ([6f10aa2](https://github.com/lmarqs/terraform-ui/commit/6f10aa2d185b8cee2ad4ce6ac2be51674f800f5d))
* separate scope plugin from context dashboard ([702db7e](https://github.com/lmarqs/terraform-ui/commit/702db7ed0637b81e687dcc8b739697291170f628))
* show active plugin name in header, keep filter active until esc ([606ac77](https://github.com/lmarqs/terraform-ui/commit/606ac77e40cce6175d493606d85237b328032f80))
* show fzf score in tree view when filtering (debug aid) ([23a3728](https://github.com/lmarqs/terraform-ui/commit/23a372821b976a4a085c7a26c655acafb5312529))
* show pinned count with pin icon in border title ([43df11f](https://github.com/lmarqs/terraform-ui/commit/43df11ffcbe141734681624b37ec2bc8e415f7cb))
* show terraform command instead of ErrReadOnly in static mode ([d25fe18](https://github.com/lmarqs/terraform-ui/commit/d25fe18293727cf7d83e1deffe344cb53e950047))
* walk module hierarchy to find nearest source location ([36ef0d5](https://github.com/lmarqs/terraform-ui/commit/36ef0d5113809d4d73f96051f53ce292fb2a403e))
* wire debug logging into app, plugins, and terraform service ([f66df93](https://github.com/lmarqs/terraform-ui/commit/f66df93451c5dbb10675c3d6add2ba78db707e88))
* wire editor integration and InputRequest system in app ([eb36f91](https://github.com/lmarqs/terraform-ui/commit/eb36f914099534f2fb84d20b66206471efc9e519))
* wire navigation stack into app layer routing and status bar ([885aade](https://github.com/lmarqs/terraform-ui/commit/885aadef8af7c2f4d90d9173424e1a91f39148aa))
* wire new bordered layout in app View() ([a20418c](https://github.com/lmarqs/terraform-ui/commit/a20418c40b215d00590ec7f054226e7c3419675c))
* wire plugin registry into app — plugins drive the TUI ([9ae851b](https://github.com/lmarqs/terraform-ui/commit/9ae851b6203d961544a07373f3945b98af59c6fe))
* wire terraform-exec service and connect TUI to real operations ([d318004](https://github.com/lmarqs/terraform-ui/commit/d3180047c54d9b2f71ce961ee18cfeb5af3de914))


### BREAKING CHANGES

* The bash version (bin/, lib/, tests/) is removed.
terraform-ui is now a Go binary with a modular plugin system.

Removed:
- bin/tfui (bash CLI)
- lib/tfui.sh (bash library)
- tests/ (BATS test suite)
- scripts/ (install.sh, package.sh)
- Dockerfile.coverage (kcov)
- Formula/ (old homebrew — now via goreleaser)
- package.json, .releaserc (semantic-release — now goreleaser)
- CONTRIBUTING.md (replaced by docs/)

Added/Changed:
- Plugin system: all features are plugins under plugins/
- Naming: "extensions" → "plugins" everywhere
- Config: tfui.yaml `plugins:` map with per-plugin config
- CI: terraform + opentofu test matrix
- Mise tasks: simplified (no go: prefix, no bash tasks)
- Slash commands: /build, /test, /lint, /fmt, /coverage, /run
- .gitignore: cleaned for Go project
- Plugin docs: docs/extensions/ (to be renamed to plugins/)

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>

# [0.39.0](https://github.com/lmarqs/terraform-ui/compare/v0.38.0...v0.39.0) (2026-05-09)


### Features

* phantom change detection and module-level grouping ([#7](https://github.com/lmarqs/terraform-ui/issues/7)) ([69af1b1](https://github.com/lmarqs/terraform-ui/commit/69af1b1f33da8a3a8d1e5089a5e9aa0642fbd918))

# [0.38.0](https://github.com/lmarqs/terraform-ui/compare/v0.37.0...v0.38.0) (2026-05-09)


### Features

* add --mode agent for structured JSON plan output ([#2](https://github.com/lmarqs/terraform-ui/issues/2)) ([be3e9d2](https://github.com/lmarqs/terraform-ui/commit/be3e9d2ba779e20f5e1bcf4cbeaa1099d6e9b858))

# [0.37.0](https://github.com/lmarqs/terraform-ui/compare/v0.36.6...v0.37.0) (2026-05-09)


### Features

* replace git-cliff with semantic-release ([42cfc81](https://github.com/lmarqs/terraform-ui/commit/42cfc81b1f36189a87006041ac308a639466ba3a))

# Changelog

## 0.36.6 — 2026-05-09

### CI

- commit VERSION back to main after release

### Miscellaneous

- v0.36.5 [skip ci]

## 0.36.5 — 2026-05-09

### Documentation

- use version = "latest" in mise install example

### Miscellaneous

- v0.36.4 [skip ci]

## 0.36.4 — 2026-05-09

### Documentation

- add mise installation method to README

### Miscellaneous

- v0.36.3 [skip ci]

## 0.36.3 — 2026-05-09

### CI

- build tarball in build task, not release

### Miscellaneous

- v0.36.2 [skip ci]

## 0.36.2 — 2026-05-09

### Bug Fixes

- preserve executable permission in release tarball

### Miscellaneous

- v0.36.1 [skip ci]

## 0.36.1 — 2026-05-09

### CI

- add git-cliff changelog generation and semantic versioning

## 0.36.0 — 2026-05-09

### Documentation

- rewrite README for clarity and add visual examples
- rewrite README with CLI reference and architecture

## 0.35.0 — 2026-05-09

### CI

- read version from artifact, not workflow outputs
- fix version to v0.35.0 (continues from v0.34.0)
- include bin/tfui and VERSION in build artifact, package as tarball
- resolve version at build step, release only consumes artifacts
- replace release-please with direct versioning from VERSION file
- only run release on push to main, skip on PRs
- grant pull-requests write permission to release job
- replace commit-count versioning with release-please

### Features

- add CLI entry point (bin/tfui)

### Refactor

- build as mise task, move syntax check to test

## 0.34.0 — 2026-05-09

### CI

- add comment explaining Node 24 env var

## 0.33.0 — 2026-05-09

### CI

- add FORCE_JAVASCRIPT_ACTIONS_TO_NODE24 to all workflows

## 0.32.0 — 2026-05-09

### CI

- opt into Node.js 24 for GitHub Actions

## 0.31.0 — 2026-05-09

### CI

- clean up release assets

## 0.30.0 — 2026-05-09

### CI

- publish lib/tfui.sh as build artifact

## 0.29.0 — 2026-05-09

### CI

- publish test and coverage reports as artifacts in releases
- replace Codecov with GitHub step summary for coverage
- fix coverage job failures
- remove release and coverage from pipeline
- fix test reporter permissions and coverage job
- add Docker-based kcov coverage runner
- add JUnit test reporting and coverage job
- restructure pipeline into main, build, test, release
- add test and release-please workflows

### Documentation

- update CLAUDE.md for BATS test workflow
- fix install.sh URL path after move to scripts/
- add project documentation and config

### Features

- add project slash commands for common workflows
- add install methods (curl, basher, homebrew)
- add tfui library

### Miscellaneous

- pin jq to major version 1
- add claude code configuration

### Refactor

- align mise tasks and slash commands to noun-verb convention
- rename commands to noun-verb convention
- rename test_helper to helpers and update references
- move install.sh and package.sh into scripts/

### Testing

- replace mock terraform with real fixtures in flow tests
- add terraform fixtures for integration testing
- remove legacy custom test framework
- migrate all scenarios to BATS test files
- add BATS framework infrastructure
- add BDD-style test suite
