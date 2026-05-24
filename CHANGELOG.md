# [1.10.0](https://github.com/lmarqs/terraform-ui/compare/v1.9.1...v1.10.0) (2026-05-24)


### Bug Fixes

* **apply:** wire AutoApprove through to macro output for CLI compat ([a0658cf](https://github.com/lmarqs/terraform-ui/commit/a0658cf4979b1dcfba11b6b81db4bc27e0d94a16))
* remove plan file deletion from exec service Apply() ([4ece192](https://github.com/lmarqs/terraform-ui/commit/4ece192711c10ac6bc6fbaa2214eed5fbd832fa3))


### Features

* **sdk:** add BackendMode type replacing *bool on InitOptions ([72b0b14](https://github.com/lmarqs/terraform-ui/commit/72b0b1474fb02c26f6623852d1ec7ab6d84ae30c))
* **sdk:** add Chdir type for relative member path ([ea8020d](https://github.com/lmarqs/terraform-ui/commit/ea8020da00e7062d9e2e1fa96a8f485e10da8c12))
* **sdk:** add PlanFile type with lifecycle-aware cleanup ([be8fd9e](https://github.com/lmarqs/terraform-ui/commit/be8fd9e604989797d84a2fdefaed065a6b2a5882))
* **sdk:** add rich domain types for Workspace, Address, LockMode, RefreshMode, DiagnosticSeverity ([6f3d4a7](https://github.com/lmarqs/terraform-ui/commit/6f3d4a7833d1ab0318fd0f55656948ce941d5b0c))

## [1.9.1](https://github.com/lmarqs/terraform-ui/compare/v1.9.0...v1.9.1) (2026-05-22)


### Bug Fixes

* **test:** update compat tapes for single-step apply confirmation ([be3947b](https://github.com/lmarqs/terraform-ui/commit/be3947b95cc7a5c7e0ee20f05b60f935f1e62379))

# [1.9.0](https://github.com/lmarqs/terraform-ui/compare/v1.8.0...v1.9.0) (2026-05-22)


### Features

* **cli:** wire --auto-approve and --target to apply plugin ([cee48e1](https://github.com/lmarqs/terraform-ui/commit/cee48e15e29c4278a6371c92574177d99966758a))

# [1.8.0](https://github.com/lmarqs/terraform-ui/compare/v1.7.3...v1.8.0) (2026-05-22)


### Bug Fixes

* add write permission to pull-requests ([f281061](https://github.com/lmarqs/terraform-ui/commit/f281061c2bdf19d0e1d7793a6e032706524f786f))
* **ci:** add top-level report job with proper permissions and checkout ([edd4e62](https://github.com/lmarqs/terraform-ui/commit/edd4e6294173fd468e126637c8fc290d2312a8f5))
* **ci:** export test reports as artifact instead of inline reporter ([9652ac3](https://github.com/lmarqs/terraform-ui/commit/9652ac3844df90a0b12d7db5165cb325d0360a33))
* **ci:** move test reporter to top-level workflow to fix check run permissions ([d81a6b3](https://github.com/lmarqs/terraform-ui/commit/d81a6b356a379b2707c189000d90c15df722218c))
* **ci:** pass pull-requests:write to build workflow caller ([eee9cfe](https://github.com/lmarqs/terraform-ui/commit/eee9cfe1fd5962ffb439f21eaae0bb9de14c688a))
* route stream messages only to active plugin ([ba080f0](https://github.com/lmarqs/terraform-ui/commit/ba080f0a34e806e618a51055e76d8467035a8eae))
* skip stack-based key routing when stack is empty ([2e8fafd](https://github.com/lmarqs/terraform-ui/commit/2e8fafdb97691af93db9cd7a79ce564610eb0c06))


### Features

* add StreamFrame for real-time terraform output ([739994f](https://github.com/lmarqs/terraform-ui/commit/739994f20c5540abd1a684a639031d4be3ffe015))
* add StreamFrame for real-time terraform output ([#12](https://github.com/lmarqs/terraform-ui/issues/12)) ([0a8592c](https://github.com/lmarqs/terraform-ui/commit/0a8592ce51742d4fea80320a949d5c8558af6701)), closes [#ci](https://github.com/lmarqs/terraform-ui/issues/ci)

## [1.7.3](https://github.com/lmarqs/terraform-ui/compare/v1.7.2...v1.7.3) (2026-05-21)


### Bug Fixes

* **ci:** pass secrets to release workflow ([063e123](https://github.com/lmarqs/terraform-ui/commit/063e12318429c26348a8d7b4b1043b30c5ff84ca))

## [1.7.2](https://github.com/lmarqs/terraform-ui/compare/v1.7.1...v1.7.2) (2026-05-21)


### Bug Fixes

* log reason for headless/interactive mode detection ([6c79196](https://github.com/lmarqs/terraform-ui/commit/6c791967399c07c1de5e8a44cdafbf633f42e0ab))

## [1.7.1](https://github.com/lmarqs/terraform-ui/compare/v1.7.0...v1.7.1) (2026-05-21)


### Bug Fixes

* serialize terraform CLI calls per directory via DirLock ([e19ccb4](https://github.com/lmarqs/terraform-ui/commit/e19ccb4921e9a25bca46859533a813de2618532d))

# [1.7.0](https://github.com/lmarqs/terraform-ui/compare/v1.6.0...v1.7.0) (2026-05-20)


### Bug Fixes

* move backend to bool flags in normalizer ([2ea39bb](https://github.com/lmarqs/terraform-ui/commit/2ea39bbdc8af63af5292f74f97ca97d0ff9a6feb))


### Features

* init plugin accepts CLI flags and implements ActivateWithArgs ([0b6015d](https://github.com/lmarqs/terraform-ui/commit/0b6015dbccd5bb9b0dda380f9583cb69fbcc580b))

# [1.6.0](https://github.com/lmarqs/terraform-ui/compare/v1.5.1...v1.6.0) (2026-05-20)


### Bug Fixes

* output filter test asserts wrong string literal ([b94bca9](https://github.com/lmarqs/terraform-ui/commit/b94bca975e698ebd268987c0a4bb7760eeda5345))
* plan filter should show list content during filtering ([b840e3a](https://github.com/lmarqs/terraform-ui/commit/b840e3ac057416552a839f8a56461b32a5ca417c))


### Features

* add batch action palette to plan plugin ([c8f5e47](https://github.com/lmarqs/terraform-ui/commit/c8f5e4742b6298b2a4d137189ee5bd1be25ac577))
* use filter symbol ([27a6f1c](https://github.com/lmarqs/terraform-ui/commit/27a6f1c63dd51c22ff0e6cae68db2a3c25a3c071))

## [1.5.1](https://github.com/lmarqs/terraform-ui/compare/v1.5.0...v1.5.1) (2026-05-19)


### Bug Fixes

* **ci:** update release workflow to use release:run task name ([825baec](https://github.com/lmarqs/terraform-ui/commit/825baec95738db71aee992522a2da04b278c3ee7))

# [1.5.0](https://github.com/lmarqs/terraform-ui/compare/v1.4.0...v1.5.0) (2026-05-19)


### Bug Fixes

* activate chdir picker on startup when no --chdir passed ([435034c](https://github.com/lmarqs/terraform-ui/commit/435034c0060660de17d470646e55230da413c31b))
* add -json flag to validate command for CLI consistency ([a440466](https://github.com/lmarqs/terraform-ui/commit/a440466759c3bdf2193c10e88fd73693806e4c8a))
* add viewport windowing to home screen menu ([0d383d6](https://github.com/lmarqs/terraform-ui/commit/0d383d6d448edd7fc4a45a7e59becef973d0be59))
* align coverage threshold to 100% across all docs and config ([89803d6](https://github.com/lmarqs/terraform-ui/commit/89803d63b34c25c5d4309fb17a8b717c2bcf156d))
* apply Busy guard to bare q key, not just :q command ([b01137b](https://github.com/lmarqs/terraform-ui/commit/b01137b4b383b2468fd0cf22299cd1ca605049f1))
* apply plugin esc handler and ctrl+r elapsed time reset ([318231a](https://github.com/lmarqs/terraform-ui/commit/318231aa7effddaae2f1f5e28709ce9651c520f0))
* broadcast result messages to all plugins, not just active ([d99cfb8](https://github.com/lmarqs/terraform-ui/commit/d99cfb8610603d9b5179adfa2cea194c81bde185))
* cancel in-flight terraform operations on plugin deactivation ([d03e7e2](https://github.com/lmarqs/terraform-ui/commit/d03e7e2b92994310f6f96f95c53a99ba3e77e693))
* **ci:** add actions:write permission for pages deploy trigger ([0270a5f](https://github.com/lmarqs/terraform-ui/commit/0270a5faeaccbff19e5bd3918c18ec4ca1cb4a5e))
* **ci:** correct binaries job dependency on renamed check job ([e5f4330](https://github.com/lmarqs/terraform-ui/commit/e5f4330990570dd98843a9a3e8aff52b806ca1a7))
* **ci:** fix release — Docker Hub publish and demo:generate arg leak ([a437e6e](https://github.com/lmarqs/terraform-ui/commit/a437e6e6c23bdf152a222975afdc1fec6f96458e))
* **ci:** grant actions:write to release caller ([bae6cb3](https://github.com/lmarqs/terraform-ui/commit/bae6cb33b1b27fba1e1920540d0ba72ef00397ba))
* **ci:** simplify release:prepare — drop redundant build and demo step ([67f9979](https://github.com/lmarqs/terraform-ui/commit/67f99798aefd6fb064dea492e4ed3850b7b64287))
* **ci:** use --quiet instead of -q for bundle install ([cd8fb48](https://github.com/lmarqs/terraform-ui/commit/cd8fb488ccf3cdcf6dd80a4340c97604893cae3b))
* clear pins when chdir changes ([1383018](https://github.com/lmarqs/terraform-ui/commit/13830186bbfdb6f3f213a7b16b8399c895514c3f))
* **cli:** init command respects --chdir, gates stderr on --ci ([afb5d76](https://github.com/lmarqs/terraform-ui/commit/afb5d7603777911151880370177a6c6cffee73c2))
* context plugin navigates to "chdir" instead of stale "scope" ID ([efaf883](https://github.com/lmarqs/terraform-ui/commit/efaf883a0081c356e6b6c98ae84ae085a222399a))
* ctrl+r in state view now bypasses cache for fresh data ([dc116dd](https://github.com/lmarqs/terraform-ui/commit/dc116dda0a69b7bc7eeb37d6843bf62b458ff7a0))
* deactivate stackable plugin when esc empties its stack ([bc53855](https://github.com/lmarqs/terraform-ui/commit/bc53855c49acefeb9f2c5389e6f1eef7409444fb))
* **demo:** auto-copy GIFs to docs/assets/demo/ in generate script ([bd02831](https://github.com/lmarqs/terraform-ui/commit/bd02831eaea386554045aea26ea2bad9f6aeda42))
* **demo:** auto-measure font cell dimensions to prevent GIF clipping ([f4354c5](https://github.com/lmarqs/terraform-ui/commit/f4354c59fa74ebe71254c471bbc4c3e786b78859))
* ensure loading feedback renders during force-unlock operation ([1fbb656](https://github.com/lmarqs/terraform-ui/commit/1fbb6562e1261272008196565362df82f226dade))
* esc during apply confirmation returns to plan ([5fdda1d](https://github.com/lmarqs/terraform-ui/commit/5fdda1d39d099b15d1842329f088ec002a8b0fac))
* esc from NavPush plugin returns to origin, not home ([b254db9](https://github.com/lmarqs/terraform-ui/commit/b254db90bfd78e0e21f0781582907e1363ea2c6f))
* esc in apply returns to plan instead of going home ([91ad80a](https://github.com/lmarqs/terraform-ui/commit/91ad80a6adf69557e484114f01e0a83901ca78c3))
* event handlers must not start async operations ([685bb04](https://github.com/lmarqs/terraform-ui/commit/685bb04c61dbf81d5eb3c2cbc156f60d71b38fcf))
* exclude timer ticks from message broadcast ([4ba022a](https://github.com/lmarqs/terraform-ui/commit/4ba022a856e2540f53e10ea19929bfd246427bf2))
* **form:** remove space-as-select from FormFrame ([64f3958](https://github.com/lmarqs/terraform-ui/commit/64f395861b89a58c0d9809b482e532bbe74ed838))
* **form:** use sdk.HintCancel constant and remove dead q handler ([e71d58b](https://github.com/lmarqs/terraform-ui/commit/e71d58b609fac61416d5b199e3896998209fc663))
* goimports formatting in cli.go ([42cd8ab](https://github.com/lmarqs/terraform-ui/commit/42cd8ab0ee956a80ad1762e71dcd9bab638d61cd))
* **init:** form submit now correctly transitions to loading state ([3fce681](https://github.com/lmarqs/terraform-ui/commit/3fce6810f980d73b6ecb4f91be30ae7e5d8e923f))
* invalidate state cache after Taint and Untaint operations ([6186017](https://github.com/lmarqs/terraform-ui/commit/61860173e2023c2f5b7fabec1182064b1a34f769))
* keep selectable arrow visible when form field is focused ([ada1f57](https://github.com/lmarqs/terraform-ui/commit/ada1f57973857c8c5ac3081926ee7c08519c106c))
* load workspaces eagerly so picker opens instantly ([4a6e543](https://github.com/lmarqs/terraform-ui/commit/4a6e543d0ba160ad2c36969e9c2b338527f8ddcd))
* **macro:** address review findings in recorder ([0b017f8](https://github.com/lmarqs/terraform-ui/commit/0b017f8919572b394776ef5b9342f6d60596a603))
* normalize all multi-char single-dash flags to double-dash ([cf5d7ac](https://github.com/lmarqs/terraform-ui/commit/cf5d7ac145b1ceae7b80c47cfe154fa2cfe1c12b))
* **pages:** add Gemfile.lock and trigger on workflow changes ([d8dcb3a](https://github.com/lmarqs/terraform-ui/commit/d8dcb3a316aed7842dee8c24c5728a93a5d84958))
* **pages:** use ruby/setup-ruby for just-the-docs theme support ([0c0858c](https://github.com/lmarqs/terraform-ui/commit/0c0858c4c9b7acc99e2d1908bf4f252b677bb91f))
* pass -target flags in apply and emit PlanInvalidatedEvent on success ([8ffe8c3](https://github.com/lmarqs/terraform-ui/commit/8ffe8c32bcfb5d85cd4d6fe19fe6b0600894d4e7))
* plan inspect frame view delegation and update macro tape ([a1c3cc4](https://github.com/lmarqs/terraform-ui/commit/a1c3cc4702f4d4b145a4fca48f9356ddfb543104))
* **plan:** preserve results on PlanInvalidatedEvent, mark stale ([6f5c275](https://github.com/lmarqs/terraform-ui/commit/6f5c27599ff48a108ce5ca73be9129bb5084191e))
* **plan:** restart timer tick chain on re-activate while loading ([3d4431f](https://github.com/lmarqs/terraform-ui/commit/3d4431f9838773a23d81fe04eddee7a27a14daa2))
* preserve chdir in header on workspace change ([5b1d98a](https://github.com/lmarqs/terraform-ui/commit/5b1d98adbd227cacd1d00941777f72745fa6219e))
* prune stale pins when plan results change ([38dd880](https://github.com/lmarqs/terraform-ui/commit/38dd880019b4a9570d623a6ff4b3544aa28e3b2c))
* remove agg from Python deps (it's a Rust binary, not a pip package) ([62a339d](https://github.com/lmarqs/terraform-ui/commit/62a339db227c81e6665230b578437df82e600810))
* remove background pre-loading, document no-magic-behavior principle ([aad3e3f](https://github.com/lmarqs/terraform-ui/commit/aad3e3fb087362aaa8df901d0f3d0e55c9f23667))
* remove double confirmation, show esc hint, reset on re-activate ([9d85d16](https://github.com/lmarqs/terraform-ui/commit/9d85d165e5a61a02fdb3bccc7ea365cf0d5b1830))
* remove redundant tfuiblast alias superseded by wildcard rule ([60195bc](https://github.com/lmarqs/terraform-ui/commit/60195bcd7784c57ea26e1a78d7535de8e4682225))
* remove unused encoding/json import from service_cache.go ([3db19a6](https://github.com/lmarqs/terraform-ui/commit/3db19a64c6eeba98cbd194140611a7447cd49fa1))
* remove unused import and dead code in ai tests ([56ae25f](https://github.com/lmarqs/terraform-ui/commit/56ae25fa68bdbe13672b62f9a88fd12e55e8299b))
* rename CLI command `workspace` to `workspaces` for consistency ([ec61285](https://github.com/lmarqs/terraform-ui/commit/ec61285905819c0b403423338e8763db69045974))
* rename header label "Scope:" to "Chdir:" to match terminology ([bdeaabd](https://github.com/lmarqs/terraform-ui/commit/bdeaabd390833e11ba628eabfe58712936568c65))
* replace single-level returnTo with multi-level navStack ([b28fd92](https://github.com/lmarqs/terraform-ui/commit/b28fd927ef4732ac2a5266619c472ff5b882bcbe))
* return to home screen after chdir selection ([2a193db](https://github.com/lmarqs/terraform-ui/commit/2a193db626e892b825b48e1208291b55f719139c))
* **sdk:** width-aware padding in scroll gutter ([38e105b](https://github.com/lmarqs/terraform-ui/commit/38e105beeb554ab31e236e428e487e4f7e78782f))
* show chdir value in header when --chdir flag is passed ([fe048e6](https://github.com/lmarqs/terraform-ui/commit/fe048e66c15ac653288a8a5cc322a57d006202ba))
* show context-appropriate hints on home screen ([c7461fa](https://github.com/lmarqs/terraform-ui/commit/c7461fae641dc7c95c4a9f6e805088fe4f4eec6b))
* show ctrl+r refresh hint in state plugin ([36ecd3e](https://github.com/lmarqs/terraform-ui/commit/36ecd3ebec7f17cf11ab903d6b0909f6c6bc7a5b))
* show loading state during force-unlock in plan and state plugins ([63496f9](https://github.com/lmarqs/terraform-ui/commit/63496f98aa80b1eadbe22edea7fb7d36df0aa55e))
* skip plan file for targeted apply (terraform incompatibility) ([60837a6](https://github.com/lmarqs/terraform-ui/commit/60837a68aa50ea8ad80e9b3272b63937e3633097))
* state/plan clear lock panel on LockClearedEvent ([3a6f50c](https://github.com/lmarqs/terraform-ui/commit/3a6f50c91d5d70aea86ba29bcae10c7e82cdbfce))
* **state:** restart timer tick chain on re-activate while loading ([b144425](https://github.com/lmarqs/terraform-ui/commit/b14442563cd6704d8f0232b90aaf21567957e3db))
* **test:** add timeout to runTfui and fix phantom field assertions ([ab09583](https://github.com/lmarqs/terraform-ui/commit/ab095838c22d85722dee22b40b703b59f6baac9c))
* **test:** simplify integration tests to terraform-only ([e845784](https://github.com/lmarqs/terraform-ui/commit/e845784127f1172bba61f0e652bfe1e067190ebe))
* **test:** tolerate exit code 2 from plan (changes present) ([3f0cfd4](https://github.com/lmarqs/terraform-ui/commit/3f0cfd4bdb4c9a91006c1365baed0d2098e8efee))
* **test:** update integration test and macro tapes for current API ([ec7bb75](https://github.com/lmarqs/terraform-ui/commit/ec7bb752b528d4ece72755a8839a9c761395e1a5))
* **ui:** add missing mock types for app coverage tests ([134e190](https://github.com/lmarqs/terraform-ui/commit/134e190a79485886ee05f9dd49f30e83e1767495))
* **ui:** clear lock and stale state on context switch ([80cb9f8](https://github.com/lmarqs/terraform-ui/commit/80cb9f8950aed4e4d0344327435a1ce61c11117f))
* **ui:** remove horizontal padding from actions bar ([0d87520](https://github.com/lmarqs/terraform-ui/commit/0d87520dcaa47efca085d4a5a90ee7fe29e97f8c))
* **ui:** respect navStack when stackable plugin's stack empties ([bf32d91](https://github.com/lmarqs/terraform-ui/commit/bf32d9119e00c43923ac619182d12612172d7cfb))
* **ui:** show project/chdir/workspace in standalone header ([0300597](https://github.com/lmarqs/terraform-ui/commit/0300597ff25356ed94213eb04e108c80e60e31f3))
* **ui:** swap standalone header positions — context left, tfui right ([81a6b88](https://github.com/lmarqs/terraform-ui/commit/81a6b8804505c7c2c8fd1aaccd7f471d008367e7))
* use display width for border title padding ([adc2d36](https://github.com/lmarqs/terraform-ui/commit/adc2d36927fbef6e4bb2bff698f34169334385b9))
* **ux:** make form submit button visually distinct ([0fcadbc](https://github.com/lmarqs/terraform-ui/commit/0fcadbc20a564dd6b877b1fe91e32bafce76e6d4))


### Features

* add :q and :q! built-in commands to command mode ([5e269b7](https://github.com/lmarqs/terraform-ui/commit/5e269b7112933e362bb7bb1ec73aa57602999fe7))
* add :version TUI plugin ([c2ea779](https://github.com/lmarqs/terraform-ui/commit/c2ea779f4b5a447bbbb9acfba3b7f8f16d064b79))
* add arch-checker agent for architectural boundary auditing ([ca42377](https://github.com/lmarqs/terraform-ui/commit/ca4237735f2b31354012b223eaaec15eaf899aa8))
* add Busy interface to SDK for quit-guard support ([e06dbb7](https://github.com/lmarqs/terraform-ui/commit/e06dbb7ea4f2c7845f97d8757f7ec05e547fc5a0))
* add elapsed time to forceunlock loading state ([99da623](https://github.com/lmarqs/terraform-ui/commit/99da623e33579f7ab348c2793204811e19c2cc6a))
* add elapsed time to output loading state ([758e4ae](https://github.com/lmarqs/terraform-ui/commit/758e4aecd2888b66f74bc90f9d6f5d50db2bc60c))
* add elapsed time to plan loading state ([2222402](https://github.com/lmarqs/terraform-ui/commit/2222402791c1e699cd3632941694948ac846812a))
* add elapsed time to state loading state ([028c880](https://github.com/lmarqs/terraform-ui/commit/028c880dd2a8adf5ce56a2239f1c19160cbfc12a))
* add elapsed time to validate loading state ([b607c95](https://github.com/lmarqs/terraform-ui/commit/b607c9528b9dad881afcebb1b771ee6feeac0625))
* add elapsed time to workspace loading state ([6d17f7a](https://github.com/lmarqs/terraform-ui/commit/6d17f7a8c10177fd3380bc409e36f2e4164bbf6d))
* add error injection to MacroService for testing ([d934355](https://github.com/lmarqs/terraform-ui/commit/d9343557ee5022ef4a92e3de4f57d73f31418070))
* add force-unlock plugin and CLI command ([bb3774a](https://github.com/lmarqs/terraform-ui/commit/bb3774a493c43b4c7a52b8afd0d0395095a748a4))
* add Header.WithWorkspace() to preserve state across updates ([8541db1](https://github.com/lmarqs/terraform-ui/commit/8541db18828d4cc294462d3e358508dbb9cf236f))
* add lock/stale badges to header with event-driven awareness ([ce78fe1](https://github.com/lmarqs/terraform-ui/commit/ce78fe1d57798598d3edfb9a4a1a67cca89dcef5))
* add LockDetected, LockCleared, StateRefreshed events to SDK ([5c323c4](https://github.com/lmarqs/terraform-ui/commit/5c323c46b7b591517a43bd01427a5c9627cf5294))
* add NavBehavior type for plugin navigation routing ([7c3aead](https://github.com/lmarqs/terraform-ui/commit/7c3aeadbf31f4d25cd4068256415222799237646))
* add NavigateMsg for inter-plugin navigation requests ([911ee19](https://github.com/lmarqs/terraform-ui/commit/911ee1929a1ff30fe434a8d47fe1a7c12988762e))
* add shared Timer component for elapsed time tracking ([78b7a31](https://github.com/lmarqs/terraform-ui/commit/78b7a311783203fa052b12badd08fd563355a736))
* add standalone import plugin ([b9067b3](https://github.com/lmarqs/terraform-ui/commit/b9067b341e6f0a0f35f518e85c272c54ad12cdd9))
* add standalone taint and untaint plugins ([13d6c85](https://github.com/lmarqs/terraform-ui/commit/13d6c852bb9eb88c5b5d52bbee202f0ae9f88880))
* add StateListOption to Service interface for cache bypass ([a3527bf](https://github.com/lmarqs/terraform-ui/commit/a3527bffa29bed9db7dd35c83cd1058bae446221))
* add t/T/A contextual keys to plan plugin ([45727e6](https://github.com/lmarqs/terraform-ui/commit/45727e63ae89b57af67c63a50b7fe21f4871f8e7))
* add tfui workspace CLI subcommands ([3d64452](https://github.com/lmarqs/terraform-ui/commit/3d64452691b1052326507748de193672a131a8c6))
* add Version() to SDK service interface ([ad9cda1](https://github.com/lmarqs/terraform-ui/commit/ad9cda14235fee69f9d0c48553395d484b8b56ce))
* add WorkspaceCreatedEvent for non-popping workspace changes ([575bd90](https://github.com/lmarqs/terraform-ui/commit/575bd9068057531e02a01ce81ea48bb9f6dc8d51))
* add WorkspaceNewOptions and WorkspaceDeleteOptions to service interface ([a061080](https://github.com/lmarqs/terraform-ui/commit/a061080cf077b4cebdd9919683343c9968b0f19a))
* apply replan for targeted resources + auto-approve ([e041b31](https://github.com/lmarqs/terraform-ui/commit/e041b311e459caa53dc09a9e87407d0f4a058642))
* **cache:** add Seed/Get/Set for outputs, diagnostics, workspaces ([dd55559](https://github.com/lmarqs/terraform-ui/commit/dd555590f6ca4711cd636e11b19d738c720eac22))
* **cli:** add --outputs, --validate-result, --workspaces flags ([4a60a07](https://github.com/lmarqs/terraform-ui/commit/4a60a07a737bd644c60a95217eeba8502d0b1b3d))
* **cli:** add tfui init subcommand ([a133f52](https://github.com/lmarqs/terraform-ui/commit/a133f529649fcad74e4c52b4e4c35a11b7fbaf53))
* **commands:** unify ux-review to cover both TUI and CLI surfaces ([84c16b3](https://github.com/lmarqs/terraform-ui/commit/84c16b36b3f66b87cb51c7bc808ffd39c3b8e2a0))
* **demo:** add 14 tape files for remaining plugins ([d540b4d](https://github.com/lmarqs/terraform-ui/commit/d540b4d09a21cbba6ab28ccb954ac8bcb8d5bbe8))
* **demo:** add demo pipeline with fixtures, tapes, and scripts ([6bd44aa](https://github.com/lmarqs/terraform-ui/commit/6bd44aa1d2bcdf765c00122233a16f983713779d))
* **demo:** add fixture files for outputs, validate, workspaces ([2483bbf](https://github.com/lmarqs/terraform-ui/commit/2483bbf436a6e2d4214c57d680f25a208d05297b))
* **demo:** batteries-included GIF generation with Python + Pillow ([cd9a1bf](https://github.com/lmarqs/terraform-ui/commit/cd9a1bf4b3142a4967504f1fb83e2d3cb85adb7e))
* **demo:** generate 14 new plugin demo GIFs ([0f37432](https://github.com/lmarqs/terraform-ui/commit/0f374324439cce999021acb95d305c9e6037485e))
* **demo:** pass new fixture flags in generate.sh ([fa5c1d0](https://github.com/lmarqs/terraform-ui/commit/fa5c1d038ae562e64801cdc5cb259347ae28d639))
* **docs:** add Ruby to mise tools and docs:* task namespace ([31e8df1](https://github.com/lmarqs/terraform-ui/commit/31e8df1f172036236fe03f321944b811b1e97870))
* enhance tfui version CLI with terraform info and -json flag ([88ec011](https://github.com/lmarqs/terraform-ui/commit/88ec011754681acb816990c11ffc67e6d9bb4634))
* **form:** allow space key to activate form fields ([b5dc2b7](https://github.com/lmarqs/terraform-ui/commit/b5dc2b72531cf520149ef8f620308b68151725f1))
* guard :q against active terraform operations ([473d9fc](https://github.com/lmarqs/terraform-ui/commit/473d9fcdb4bfca8fbc10146a72c4dfb89db1258f))
* handle NavigateMsg in app router ([b8062a2](https://github.com/lmarqs/terraform-ui/commit/b8062a2daaa3b518e467fafd9f0b83b98d9972a4))
* implement Busy interface on apply, plan, and state plugins ([f3f5992](https://github.com/lmarqs/terraform-ui/commit/f3f59929b724ae176f31e8a0cbbd1690121af9c4))
* init plugin uses member blocks, adds --force flag ([3a1c8d5](https://github.com/lmarqs/terraform-ui/commit/3a1c8d58821d4a74801accc55775934cfeca17c6))
* **macro:** add --record flag for session frame capture ([3329901](https://github.com/lmarqs/terraform-ui/commit/332990189d38a51a1a87173b4fcf2de897886414))
* **macro:** read outputs/diagnostics/workspaces from cache ([350524c](https://github.com/lmarqs/terraform-ui/commit/350524ca9c308415abeb2f9f7e4988f45c7a5add))
* navigate back to previous plugin after chdir/workspace selection ([0792446](https://github.com/lmarqs/terraform-ui/commit/0792446b0940c4b9b63254545c4dd904f39570e6))
* plan plugin UX overhaul — tree view, filter, inspect, wrap/pan ([3173c44](https://github.com/lmarqs/terraform-ui/commit/3173c44723fe4648817542cabacede50cce73a53))
* **plan:** add `e` (edit) keybinding to open source file ([f60ef4b](https://github.com/lmarqs/terraform-ui/commit/f60ef4b771b3c9c0f792d42b20a9b479ce88c90b))
* **plugins:** add actions bar to detail views, integrate workspace ([a21c79b](https://github.com/lmarqs/terraform-ui/commit/a21c79bdb6287ec0ab0cc1564e3fd1d8569d0aa9))
* **plugins:** add back hint to force-unlock plugin ([dbb7f45](https://github.com/lmarqs/terraform-ui/commit/dbb7f455fa09c3d8d788bc1f7bad3b4640edfcb6))
* **plugins:** add Positionable to blastradius, phantom, risk, output ([9362c12](https://github.com/lmarqs/terraform-ui/commit/9362c123cc8d548fc1c0c9a6d8903ebe8bcd0e77))
* **plugins:** implement Outputter on apply, validate, output, version, init ([2a183ab](https://github.com/lmarqs/terraform-ui/commit/2a183abe2077021df352fc477e5679d70c75cf0a))
* **plugins:** implement Outputter on plan and state plugins ([d4f94c8](https://github.com/lmarqs/terraform-ui/commit/d4f94c8f825fb24e2f370cd0cb0304b7ab60e9ca))
* **plugins:** integrate actions bar and scroll gutter in state/plan ([7be5ccb](https://github.com/lmarqs/terraform-ui/commit/7be5ccbf999d53285450771afede356373c3526a))
* **plugins:** show force-unlock action chip in lock error state ([101e124](https://github.com/lmarqs/terraform-ui/commit/101e124b472aa5b4adaeafb404a61fa3eb87c69d))
* re-resolve config on workspace and chdir changes ([b2c8e2f](https://github.com/lmarqs/terraform-ui/commit/b2c8e2fd5bbb0852d2b5426909ba674e839f818b))
* rename repl plugin to console, fix key capture ([5cba792](https://github.com/lmarqs/terraform-ui/commit/5cba792496c62bc2dbbe5c5dcbcf83ce4560d7fb))
* **sdk:** add ActionsBar, ScrollGutter primitives and Positionable interface ([114c022](https://github.com/lmarqs/terraform-ui/commit/114c022f25a9b84f73bd70091e454242fbf96b9f))
* **sdk:** add InitOptions to Service.Init interface ([0026422](https://github.com/lmarqs/terraform-ui/commit/0026422cf7823a0572062f1c10fcda33fb76dcf2))
* **sdk:** add KeyCapturer interface, rename InputModeREPL to InputModeConsole ([1dd8a46](https://github.com/lmarqs/terraform-ui/commit/1dd8a465585cb6b45041aa0e01b5fd0307a397d7))
* **sdk:** add Outputter, ExitCoder, and ActivateWithArgs interfaces ([4365915](https://github.com/lmarqs/terraform-ui/commit/4365915e5c8000ecc18bc0aef91816c5a9b147b9))
* **sdktest:** add shared MockService for plugin tests ([3a72a32](https://github.com/lmarqs/terraform-ui/commit/3a72a32cffb1002a8dd0bb2e2ae8cea0efc75f3c))
* show tainted indicator on resources in state view ([b1df262](https://github.com/lmarqs/terraform-ui/commit/b1df26218b342c743255b259f440c986b3b3f438))
* **skills:** add cli-design skill for parity with tui-design ([7258bfd](https://github.com/lmarqs/terraform-ui/commit/7258bfda28b8fbd7b123d95073e19c46a251d61b))
* **tui:** add init plugin with form-driven UI ([00924b2](https://github.com/lmarqs/terraform-ui/commit/00924b2f759a8c5053e2a349589380a7956df2e5))
* **ui:** add standalone mode to App model ([1f078b9](https://github.com/lmarqs/terraform-ui/commit/1f078b951ca75bf40beb941041a8838c339fb456))
* **ui:** wire position counter in title bar and strip ↑↓ from home hints ([10e019d](https://github.com/lmarqs/terraform-ui/commit/10e019dcf009e5f1f49241bbc4e73185c72d0e31))
* wire taint/untaint/import/auto-approve navigation in app ([241deca](https://github.com/lmarqs/terraform-ui/commit/241deca479e3cd217b462dd19dfdacba400437c7))


### Reverts

* undo incorrect workspace→workspaces CLI rename ([4f5bc3d](https://github.com/lmarqs/terraform-ui/commit/4f5bc3d171b62cc1b6f904998e862af7a99ac6dd))

# [1.4.0](https://github.com/lmarqs/terraform-ui/compare/v1.3.1...v1.4.0) (2026-05-14)


### Bug Fixes

* enforce macro safety contract and remove --plan/--state requirement ([e543666](https://github.com/lmarqs/terraform-ui/commit/e543666c492aac99f871a73a498de6b504a37bd2))
* update add-plugin command and ux-checker agent for pkg/sdk ([539df9f](https://github.com/lmarqs/terraform-ui/commit/539df9f8dfd10c4df59dacfa5cee55fe0c412c0e))


### Features

* add ServiceCache for typed, source-aware terraform data caching ([6e1e59f](https://github.com/lmarqs/terraform-ui/commit/6e1e59f82e779adc1938c02ef47e53dafe2a56e8))
* **cli:** wire CompositeService into TUI and macro modes ([759ce89](https://github.com/lmarqs/terraform-ui/commit/759ce89a304fa97324f46192cc0afd2b16898e28))
* **terraform:** add statePath support to TerraformService ([517db0f](https://github.com/lmarqs/terraform-ui/commit/517db0f197905f7ed74a6789d35e672275cb1072))
* **terraform:** implement CompositeService for hybrid read/write mode ([5b68f7d](https://github.com/lmarqs/terraform-ui/commit/5b68f7d2a6abba925c0ef54c5c5a25881c838dfe))

## [1.3.1](https://github.com/lmarqs/terraform-ui/compare/v1.3.0...v1.3.1) (2026-05-13)


### Bug Fixes

* **ci:** intentionally break publishCmd to verify CI failure reporting ([c78caf3](https://github.com/lmarqs/terraform-ui/commit/c78caf32f15fdb8ef2462305cda440a5055bac67))
* **ci:** restore goreleaser publishCmd after CI failure verification ([3023c1a](https://github.com/lmarqs/terraform-ui/commit/3023c1af621d4b6b73f2eec483930c7a02f06802))

## [1.3.1](https://github.com/lmarqs/terraform-ui/compare/v1.3.0...v1.3.1) (2026-05-13)


### Bug Fixes

* **ci:** intentionally break publishCmd to verify CI failure reporting ([c78caf3](https://github.com/lmarqs/terraform-ui/commit/c78caf32f15fdb8ef2462305cda440a5055bac67))

# [1.3.0](https://github.com/lmarqs/terraform-ui/compare/v1.2.0...v1.3.0) (2026-05-13)


### Bug Fixes

* **ci:** disable corrupted Go cache and pre-download modules in release ([d9f7196](https://github.com/lmarqs/terraform-ui/commit/d9f7196a08bf1d1b8f7866ede66e2f9c8bf93a09))
* **ci:** remove orphan release tags to unblock publishing ([75390f4](https://github.com/lmarqs/terraform-ui/commit/75390f4651e1c72080948bd807f85ec13ce0e333))
* **ci:** update workflow to use renamed integration task ([f1ce78f](https://github.com/lmarqs/terraform-ui/commit/f1ce78f42901a40b3f0e0a8442346931fbe99731))
* **ci:** use mise-managed Go in release instead of actions/setup-go ([e7f98f0](https://github.com/lmarqs/terraform-ui/commit/e7f98f03996413a7e078e1c23ed1f55e4b507c1e))
* init plugin generates HCL to tfui.hcl instead of YAML ([d055e77](https://github.com/lmarqs/terraform-ui/commit/d055e771c39ad9b2b6862ba52df3d5ba4d4d41aa))
* process pending commands in macro WaitUntil loop ([d4f3f03](https://github.com/lmarqs/terraform-ui/commit/d4f3f03e40bbae422c8e96d3b8304d6c58818e5b))
* re-enable GPG signing in release task ([ac2d8ed](https://github.com/lmarqs/terraform-ui/commit/ac2d8edb9627b65ebbf05188e9db31d0ddd27b59))
* remove @semantic-release/github plugin (goreleaser owns the release) ([e9c1bf6](https://github.com/lmarqs/terraform-ui/commit/e9c1bf63d9680cd6ec33b9e49e05d03d8d9975f1))
* remove command filtering from RecordingService ([045003a](https://github.com/lmarqs/terraform-ui/commit/045003a94e032a5b03f60f9332809fca3506611f))
* replace StdinProvider consumed bool with sync.Once ([cd784f1](https://github.com/lmarqs/terraform-ui/commit/cd784f178d1246ccaa6a39beb4a93e02e09a53ab))
* show targeted resource count in apply confirmation when pins exist ([cde8fef](https://github.com/lmarqs/terraform-ui/commit/cde8feffdbc53b5d3822b20f4863b776c08534aa))
* **test:** correct apply_targeted.tape wait text for pinned resources ([e7a4396](https://github.com/lmarqs/terraform-ui/commit/e7a4396cf20b5144d2098038ca4662530cf58e73))
* **test:** correct targeted apply wait text in integration test ([83cc809](https://github.com/lmarqs/terraform-ui/commit/83cc80968693c4d601a39b3f485bf009dedcda45))
* use go-isatty for portable TTY detection ([5396a9b](https://github.com/lmarqs/terraform-ui/commit/5396a9b4ef13bb2821d87ea4075d25ec581bac7b))


### Features

* add argument normalizer and wire HCL config in CLI ([ef7795e](https://github.com/lmarqs/terraform-ui/commit/ef7795efc2b4e8579b380d29509fb24f5840cdc4))
* add exploratory-tester agent for macro-driven smoke testing ([5d29746](https://github.com/lmarqs/terraform-ui/commit/5d29746eb48a263d966c111ec92010d528479e81))
* add tofu and terragrunt to mise, per-binary integration tasks ([cb47504](https://github.com/lmarqs/terraform-ui/commit/cb47504300e3214ae0ec5b6ea042140a62a01c7d))
* add typed event bus, replace session-key polling with reactive handlers ([306091a](https://github.com/lmarqs/terraform-ui/commit/306091abda0888ce3e5bf926c9b5b79974e7a9a2))
* change Service interface to PlanOptions/ApplyOptions ([161f061](https://github.com/lmarqs/terraform-ui/commit/161f061c1acc3c898aac511cdd64c043afe2dd16))
* implement HCL config parsing (LoadRoot, LoadChild, Resolve) ([9c42fa5](https://github.com/lmarqs/terraform-ui/commit/9c42fa5803d60da3818644c6cadb05922f29eb71))
* live workspace re-resolve and remove all YAML vestiges ([bcb207c](https://github.com/lmarqs/terraform-ui/commit/bcb207c2e094a02f8f3e8318cfa6161ca8ad50c9))
* macro mode outputs terraform commands to stdout ([a1e6375](https://github.com/lmarqs/terraform-ui/commit/a1e6375602ff8da3c124c69837a6596491e8bfb8))
* record all operations consistently (no read/write distinction) ([f4073b4](https://github.com/lmarqs/terraform-ui/commit/f4073b42eb484d51d5c942766edfc3623095fa44))
* remove @semantic-release/github dependency ([b6696ea](https://github.com/lmarqs/terraform-ui/commit/b6696ea9599651f3eb98049ad21154ac0d99b9c9))
* replace scope plugin with chdir plugin ([fb53be2](https://github.com/lmarqs/terraform-ui/commit/fb53be2be998a2baf456c29de2d37e3be7722eaa))
* wire -- passthrough to ExtraArgs in PlanOptions/ApplyOptions ([2450b28](https://github.com/lmarqs/terraform-ui/commit/2450b2833dcf8e20b13ca99be11589faef65484e))

# [1.3.0](https://github.com/lmarqs/terraform-ui/compare/v1.2.0...v1.3.0) (2026-05-13)


### Bug Fixes

* **ci:** disable corrupted Go cache and pre-download modules in release ([d9f7196](https://github.com/lmarqs/terraform-ui/commit/d9f7196a08bf1d1b8f7866ede66e2f9c8bf93a09))
* **ci:** remove orphan release tags to unblock publishing ([75390f4](https://github.com/lmarqs/terraform-ui/commit/75390f4651e1c72080948bd807f85ec13ce0e333))
* **ci:** update workflow to use renamed integration task ([f1ce78f](https://github.com/lmarqs/terraform-ui/commit/f1ce78f42901a40b3f0e0a8442346931fbe99731))
* init plugin generates HCL to tfui.hcl instead of YAML ([d055e77](https://github.com/lmarqs/terraform-ui/commit/d055e771c39ad9b2b6862ba52df3d5ba4d4d41aa))
* process pending commands in macro WaitUntil loop ([d4f3f03](https://github.com/lmarqs/terraform-ui/commit/d4f3f03e40bbae422c8e96d3b8304d6c58818e5b))
* re-enable GPG signing in release task ([ac2d8ed](https://github.com/lmarqs/terraform-ui/commit/ac2d8edb9627b65ebbf05188e9db31d0ddd27b59))
* remove @semantic-release/github plugin (goreleaser owns the release) ([e9c1bf6](https://github.com/lmarqs/terraform-ui/commit/e9c1bf63d9680cd6ec33b9e49e05d03d8d9975f1))
* remove command filtering from RecordingService ([045003a](https://github.com/lmarqs/terraform-ui/commit/045003a94e032a5b03f60f9332809fca3506611f))
* replace StdinProvider consumed bool with sync.Once ([cd784f1](https://github.com/lmarqs/terraform-ui/commit/cd784f178d1246ccaa6a39beb4a93e02e09a53ab))
* show targeted resource count in apply confirmation when pins exist ([cde8fef](https://github.com/lmarqs/terraform-ui/commit/cde8feffdbc53b5d3822b20f4863b776c08534aa))
* **test:** correct apply_targeted.tape wait text for pinned resources ([e7a4396](https://github.com/lmarqs/terraform-ui/commit/e7a4396cf20b5144d2098038ca4662530cf58e73))
* **test:** correct targeted apply wait text in integration test ([83cc809](https://github.com/lmarqs/terraform-ui/commit/83cc80968693c4d601a39b3f485bf009dedcda45))
* use go-isatty for portable TTY detection ([5396a9b](https://github.com/lmarqs/terraform-ui/commit/5396a9b4ef13bb2821d87ea4075d25ec581bac7b))


### Features

* add argument normalizer and wire HCL config in CLI ([ef7795e](https://github.com/lmarqs/terraform-ui/commit/ef7795efc2b4e8579b380d29509fb24f5840cdc4))
* add exploratory-tester agent for macro-driven smoke testing ([5d29746](https://github.com/lmarqs/terraform-ui/commit/5d29746eb48a263d966c111ec92010d528479e81))
* add tofu and terragrunt to mise, per-binary integration tasks ([cb47504](https://github.com/lmarqs/terraform-ui/commit/cb47504300e3214ae0ec5b6ea042140a62a01c7d))
* add typed event bus, replace session-key polling with reactive handlers ([306091a](https://github.com/lmarqs/terraform-ui/commit/306091abda0888ce3e5bf926c9b5b79974e7a9a2))
* change Service interface to PlanOptions/ApplyOptions ([161f061](https://github.com/lmarqs/terraform-ui/commit/161f061c1acc3c898aac511cdd64c043afe2dd16))
* implement HCL config parsing (LoadRoot, LoadChild, Resolve) ([9c42fa5](https://github.com/lmarqs/terraform-ui/commit/9c42fa5803d60da3818644c6cadb05922f29eb71))
* live workspace re-resolve and remove all YAML vestiges ([bcb207c](https://github.com/lmarqs/terraform-ui/commit/bcb207c2e094a02f8f3e8318cfa6161ca8ad50c9))
* macro mode outputs terraform commands to stdout ([a1e6375](https://github.com/lmarqs/terraform-ui/commit/a1e6375602ff8da3c124c69837a6596491e8bfb8))
* record all operations consistently (no read/write distinction) ([f4073b4](https://github.com/lmarqs/terraform-ui/commit/f4073b42eb484d51d5c942766edfc3623095fa44))
* remove @semantic-release/github dependency ([b6696ea](https://github.com/lmarqs/terraform-ui/commit/b6696ea9599651f3eb98049ad21154ac0d99b9c9))
* replace scope plugin with chdir plugin ([fb53be2](https://github.com/lmarqs/terraform-ui/commit/fb53be2be998a2baf456c29de2d37e3be7722eaa))
* wire -- passthrough to ExtraArgs in PlanOptions/ApplyOptions ([2450b28](https://github.com/lmarqs/terraform-ui/commit/2450b2833dcf8e20b13ca99be11589faef65484e))

# [1.3.0](https://github.com/lmarqs/terraform-ui/compare/v1.2.0...v1.3.0) (2026-05-13)


### Bug Fixes

* **ci:** remove orphan release tags to unblock publishing ([75390f4](https://github.com/lmarqs/terraform-ui/commit/75390f4651e1c72080948bd807f85ec13ce0e333))
* **ci:** update workflow to use renamed integration task ([f1ce78f](https://github.com/lmarqs/terraform-ui/commit/f1ce78f42901a40b3f0e0a8442346931fbe99731))
* init plugin generates HCL to tfui.hcl instead of YAML ([d055e77](https://github.com/lmarqs/terraform-ui/commit/d055e771c39ad9b2b6862ba52df3d5ba4d4d41aa))
* process pending commands in macro WaitUntil loop ([d4f3f03](https://github.com/lmarqs/terraform-ui/commit/d4f3f03e40bbae422c8e96d3b8304d6c58818e5b))
* re-enable GPG signing in release task ([ac2d8ed](https://github.com/lmarqs/terraform-ui/commit/ac2d8edb9627b65ebbf05188e9db31d0ddd27b59))
* remove @semantic-release/github plugin (goreleaser owns the release) ([e9c1bf6](https://github.com/lmarqs/terraform-ui/commit/e9c1bf63d9680cd6ec33b9e49e05d03d8d9975f1))
* remove command filtering from RecordingService ([045003a](https://github.com/lmarqs/terraform-ui/commit/045003a94e032a5b03f60f9332809fca3506611f))
* replace StdinProvider consumed bool with sync.Once ([cd784f1](https://github.com/lmarqs/terraform-ui/commit/cd784f178d1246ccaa6a39beb4a93e02e09a53ab))
* show targeted resource count in apply confirmation when pins exist ([cde8fef](https://github.com/lmarqs/terraform-ui/commit/cde8feffdbc53b5d3822b20f4863b776c08534aa))
* **test:** correct apply_targeted.tape wait text for pinned resources ([e7a4396](https://github.com/lmarqs/terraform-ui/commit/e7a4396cf20b5144d2098038ca4662530cf58e73))
* **test:** correct targeted apply wait text in integration test ([83cc809](https://github.com/lmarqs/terraform-ui/commit/83cc80968693c4d601a39b3f485bf009dedcda45))
* use go-isatty for portable TTY detection ([5396a9b](https://github.com/lmarqs/terraform-ui/commit/5396a9b4ef13bb2821d87ea4075d25ec581bac7b))


### Features

* add argument normalizer and wire HCL config in CLI ([ef7795e](https://github.com/lmarqs/terraform-ui/commit/ef7795efc2b4e8579b380d29509fb24f5840cdc4))
* add exploratory-tester agent for macro-driven smoke testing ([5d29746](https://github.com/lmarqs/terraform-ui/commit/5d29746eb48a263d966c111ec92010d528479e81))
* add tofu and terragrunt to mise, per-binary integration tasks ([cb47504](https://github.com/lmarqs/terraform-ui/commit/cb47504300e3214ae0ec5b6ea042140a62a01c7d))
* add typed event bus, replace session-key polling with reactive handlers ([306091a](https://github.com/lmarqs/terraform-ui/commit/306091abda0888ce3e5bf926c9b5b79974e7a9a2))
* change Service interface to PlanOptions/ApplyOptions ([161f061](https://github.com/lmarqs/terraform-ui/commit/161f061c1acc3c898aac511cdd64c043afe2dd16))
* implement HCL config parsing (LoadRoot, LoadChild, Resolve) ([9c42fa5](https://github.com/lmarqs/terraform-ui/commit/9c42fa5803d60da3818644c6cadb05922f29eb71))
* live workspace re-resolve and remove all YAML vestiges ([bcb207c](https://github.com/lmarqs/terraform-ui/commit/bcb207c2e094a02f8f3e8318cfa6161ca8ad50c9))
* macro mode outputs terraform commands to stdout ([a1e6375](https://github.com/lmarqs/terraform-ui/commit/a1e6375602ff8da3c124c69837a6596491e8bfb8))
* record all operations consistently (no read/write distinction) ([f4073b4](https://github.com/lmarqs/terraform-ui/commit/f4073b42eb484d51d5c942766edfc3623095fa44))
* remove @semantic-release/github dependency ([b6696ea](https://github.com/lmarqs/terraform-ui/commit/b6696ea9599651f3eb98049ad21154ac0d99b9c9))
* replace scope plugin with chdir plugin ([fb53be2](https://github.com/lmarqs/terraform-ui/commit/fb53be2be998a2baf456c29de2d37e3be7722eaa))
* wire -- passthrough to ExtraArgs in PlanOptions/ApplyOptions ([2450b28](https://github.com/lmarqs/terraform-ui/commit/2450b2833dcf8e20b13ca99be11589faef65484e))

# [1.6.0](https://github.com/lmarqs/terraform-ui/compare/v1.5.0...v1.6.0) (2026-05-13)


### Bug Fixes

* **ci:** update workflow to use renamed integration task ([f1ce78f](https://github.com/lmarqs/terraform-ui/commit/f1ce78f42901a40b3f0e0a8442346931fbe99731))
* init plugin generates HCL to tfui.hcl instead of YAML ([d055e77](https://github.com/lmarqs/terraform-ui/commit/d055e771c39ad9b2b6862ba52df3d5ba4d4d41aa))
* process pending commands in macro WaitUntil loop ([d4f3f03](https://github.com/lmarqs/terraform-ui/commit/d4f3f03e40bbae422c8e96d3b8304d6c58818e5b))
* remove command filtering from RecordingService ([045003a](https://github.com/lmarqs/terraform-ui/commit/045003a94e032a5b03f60f9332809fca3506611f))
* replace StdinProvider consumed bool with sync.Once ([cd784f1](https://github.com/lmarqs/terraform-ui/commit/cd784f178d1246ccaa6a39beb4a93e02e09a53ab))
* show targeted resource count in apply confirmation when pins exist ([cde8fef](https://github.com/lmarqs/terraform-ui/commit/cde8feffdbc53b5d3822b20f4863b776c08534aa))
* **test:** correct apply_targeted.tape wait text for pinned resources ([e7a4396](https://github.com/lmarqs/terraform-ui/commit/e7a4396cf20b5144d2098038ca4662530cf58e73))
* **test:** correct targeted apply wait text in integration test ([83cc809](https://github.com/lmarqs/terraform-ui/commit/83cc80968693c4d601a39b3f485bf009dedcda45))
* use go-isatty for portable TTY detection ([5396a9b](https://github.com/lmarqs/terraform-ui/commit/5396a9b4ef13bb2821d87ea4075d25ec581bac7b))


### Features

* add argument normalizer and wire HCL config in CLI ([ef7795e](https://github.com/lmarqs/terraform-ui/commit/ef7795efc2b4e8579b380d29509fb24f5840cdc4))
* add exploratory-tester agent for macro-driven smoke testing ([5d29746](https://github.com/lmarqs/terraform-ui/commit/5d29746eb48a263d966c111ec92010d528479e81))
* add tofu and terragrunt to mise, per-binary integration tasks ([cb47504](https://github.com/lmarqs/terraform-ui/commit/cb47504300e3214ae0ec5b6ea042140a62a01c7d))
* add typed event bus, replace session-key polling with reactive handlers ([306091a](https://github.com/lmarqs/terraform-ui/commit/306091abda0888ce3e5bf926c9b5b79974e7a9a2))
* change Service interface to PlanOptions/ApplyOptions ([161f061](https://github.com/lmarqs/terraform-ui/commit/161f061c1acc3c898aac511cdd64c043afe2dd16))
* implement HCL config parsing (LoadRoot, LoadChild, Resolve) ([9c42fa5](https://github.com/lmarqs/terraform-ui/commit/9c42fa5803d60da3818644c6cadb05922f29eb71))
* live workspace re-resolve and remove all YAML vestiges ([bcb207c](https://github.com/lmarqs/terraform-ui/commit/bcb207c2e094a02f8f3e8318cfa6161ca8ad50c9))
* replace scope plugin with chdir plugin ([fb53be2](https://github.com/lmarqs/terraform-ui/commit/fb53be2be998a2baf456c29de2d37e3be7722eaa))
* wire -- passthrough to ExtraArgs in PlanOptions/ApplyOptions ([2450b28](https://github.com/lmarqs/terraform-ui/commit/2450b2833dcf8e20b13ca99be11589faef65484e))

# [1.5.0](https://github.com/lmarqs/terraform-ui/compare/v1.4.0...v1.5.0) (2026-05-12)


### Features

* record all operations consistently (no read/write distinction) ([f4073b4](https://github.com/lmarqs/terraform-ui/commit/f4073b42eb484d51d5c942766edfc3623095fa44))

# [1.4.0](https://github.com/lmarqs/terraform-ui/compare/v1.3.0...v1.4.0) (2026-05-12)


### Features

* macro mode outputs terraform commands to stdout ([a1e6375](https://github.com/lmarqs/terraform-ui/commit/a1e6375602ff8da3c124c69837a6596491e8bfb8))

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
