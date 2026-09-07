# Changelog

## [3.25.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.24.0...v3.25.0) (2026-09-07)


### Features

* add stream_settings (incl. tls_settings) to xray_outbounds ([#455](https://github.com/batonogov/terraform-provider-threexui/issues/455)) ([58b9b72](https://github.com/batonogov/terraform-provider-threexui/commit/58b9b72330b6c01dc9268311ae31f0a5b668de95))

## [3.24.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.23.1...v3.24.0) (2026-08-25)


### Features

* support the AmneziaWG inbound protocol ([#448](https://github.com/batonogov/terraform-provider-threexui/issues/448)) ([6b38b80](https://github.com/batonogov/terraform-provider-threexui/commit/6b38b80e2ecb96a7c53e5671041e0c876a8a35c2))


### Bug Fixes

* free peer emails when deleting a WireGuard/AmneziaWG inbound ([#454](https://github.com/batonogov/terraform-provider-threexui/issues/454)) ([33e0359](https://github.com/batonogov/terraform-provider-threexui/commit/33e0359d99c4e78df61aa8c0f847d7e1c564cb61))
* restart the panel for notifier settings read at startup ([#451](https://github.com/batonogov/terraform-provider-threexui/issues/451)) ([bf9f902](https://github.com/batonogov/terraform-provider-threexui/commit/bf9f90297113a7b1fdee462e2d3c5ec85af4aefc))

## [3.23.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.23.0...v3.23.1) (2026-08-25)


### Bug Fixes

* deprecate threexui_inbound.all_time ([#446](https://github.com/batonogov/terraform-provider-threexui/issues/446)) ([bc343eb](https://github.com/batonogov/terraform-provider-threexui/commit/bc343eb1757481d0d3719354de7f059044f86316))
* restart the panel for every startup-read subscription setting ([#447](https://github.com/batonogov/terraform-provider-threexui/issues/447)) ([b42ed0c](https://github.com/batonogov/terraform-provider-threexui/commit/b42ed0c8c8609ca6e4a6b191815990509024ab2a))

## [3.23.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.22.3...v3.23.0) (2026-08-25)


### Features

* support 3x-ui v3.7.0 and drop the 3.1.x line ([#440](https://github.com/batonogov/terraform-provider-threexui/issues/440)) ([010889f](https://github.com/batonogov/terraform-provider-threexui/commit/010889f07e48f4ecf0d827e0c83b3a87bb424776))

## [3.22.3](https://github.com/batonogov/terraform-provider-threexui/compare/v3.22.2...v3.22.3) (2026-08-15)


### Bug Fixes

* complete v3.6.0 version-addition procedure (contract, drift gate, docs) ([#435](https://github.com/batonogov/terraform-provider-threexui/issues/435)) ([7eb9678](https://github.com/batonogov/terraform-provider-threexui/commit/7eb96786e7ee8f9113606e110320c6584f5404f2))

## [3.22.2](https://github.com/batonogov/terraform-provider-threexui/compare/v3.22.1...v3.22.2) (2026-08-05)


### Bug Fixes

* default balancer baselines to typed null ([#423](https://github.com/batonogov/terraform-provider-threexui/issues/423)) ([4714f7c](https://github.com/batonogov/terraform-provider-threexui/commit/4714f7cb189ed20c4a173a70b01620f2915f5a46))
* keep DNS server fields with configured entries ([#425](https://github.com/batonogov/terraform-provider-threexui/issues/425)) ([7997614](https://github.com/batonogov/terraform-provider-threexui/commit/79976146b47ff2163bc82f36830bb8306771c9c5))
* keep omitted xray basics log absent ([#422](https://github.com/batonogov/terraform-provider-threexui/issues/422)) ([0e4c7f7](https://github.com/batonogov/terraform-provider-threexui/commit/0e4c7f7753de50c3b74244ca68ad119773aec57d)), closes [#208](https://github.com/batonogov/terraform-provider-threexui/issues/208)
* **node:** handle unknown inbound tags in planning ([#424](https://github.com/batonogov/terraform-provider-threexui/issues/424)) ([f7d1b10](https://github.com/batonogov/terraform-provider-threexui/commit/f7d1b107b6dc6c7949128799957b131408183b2a))
* prevent xray outbound field bleed ([#426](https://github.com/batonogov/terraform-provider-threexui/issues/426)) ([e6419c5](https://github.com/batonogov/terraform-provider-threexui/commit/e6419c51e52a146bed34305ae57d7ef273f157dc))
* reconcile xray policy levels by id ([#427](https://github.com/batonogov/terraform-provider-threexui/issues/427)) ([372a184](https://github.com/batonogov/terraform-provider-threexui/commit/372a18491ba7f6197a4a61f27c61cb5d3259835a))

## [3.22.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.22.0...v3.22.1) (2026-08-05)


### Bug Fixes

* prevent routing rule field bleed when rules are removed or reordered ([#415](https://github.com/batonogov/terraform-provider-threexui/issues/415)) ([7b8aa6e](https://github.com/batonogov/terraform-provider-threexui/commit/7b8aa6e079c734948e8ce94d0bcd0a0f9011d1ff))

## [3.22.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.21.0...v3.22.0) (2026-07-30)


### Features

* support 3x-ui v3.6.0 ([#411](https://github.com/batonogov/terraform-provider-threexui/issues/411)) ([a858299](https://github.com/batonogov/terraform-provider-threexui/commit/a858299ecedfca199d556802d8ba2ba091c7ea26))

## [3.21.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.20.1...v3.21.0) (2026-07-27)


### Features

* expose REALITY minClientVer/maxClientVer/maxTimediff on reality_settings ([#409](https://github.com/batonogov/terraform-provider-threexui/issues/409)) ([14f9b40](https://github.com/batonogov/terraform-provider-threexui/commit/14f9b40572d777ffe5b32e25cc458c43894f8141))

## [3.20.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.20.0...v3.20.1) (2026-07-21)


### Bug Fixes

* align schema security contracts ([#399](https://github.com/batonogov/terraform-provider-threexui/issues/399)) ([69fcc45](https://github.com/batonogov/terraform-provider-threexui/commit/69fcc45e71b90f5e25ac3d97d356688faeea9643))

## [3.20.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.19.1...v3.20.0) (2026-07-15)


### Features

* add typed mtproto_settings block to threexui_inbound ([#335](https://github.com/batonogov/terraform-provider-threexui/issues/335)) ([#361](https://github.com/batonogov/terraform-provider-threexui/issues/361)) ([467bf2d](https://github.com/batonogov/terraform-provider-threexui/commit/467bf2d23105c0aa05e0fd51e9f5e6222b87e870))
* add write-only variants for inbound_client password/secret ([#358](https://github.com/batonogov/terraform-provider-threexui/issues/358)) ([2a7c444](https://github.com/batonogov/terraform-provider-threexui/commit/2a7c444c9099b8d71a3dd5b806c96ec993713f2d))
* expose xray Observatory/BurstObservatory as typed resource ([#362](https://github.com/batonogov/terraform-provider-threexui/issues/362)) ([45eebe4](https://github.com/batonogov/terraform-provider-threexui/commit/45eebe49c013dbad9113707dc59a8512fb2c5b08))


### Bug Fixes

* acquire inboundClientMu in InboundResource to close race with inbound_client ([#343](https://github.com/batonogov/terraform-provider-threexui/issues/343)) ([#356](https://github.com/batonogov/terraform-provider-threexui/issues/356)) ([52b1a4d](https://github.com/batonogov/terraform-provider-threexui/commit/52b1a4dfd2dd8c58645b7d5a31629ac1fbb88b49))
* add defensive hysteria2 back-compat for hysteria_settings retention ([#341](https://github.com/batonogov/terraform-provider-threexui/issues/341)) ([#355](https://github.com/batonogov/terraform-provider-threexui/issues/355)) ([ff6e6dc](https://github.com/batonogov/terraform-provider-threexui/commit/ff6e6dc3d17693c3f469704aa8baf876be9e848b))
* backfill missing attrs in docs, add Deprecated/Sensitive labels ([#344](https://github.com/batonogov/terraform-provider-threexui/issues/344)) ([#357](https://github.com/batonogov/terraform-provider-threexui/issues/357)) ([ddc8829](https://github.com/batonogov/terraform-provider-threexui/commit/ddc882922fd6e514c3790407142ddc8cd2bccffa))

## [3.19.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.19.0...v3.19.1) (2026-07-14)


### Bug Fixes

* audit easy-wins — hysteria drift, auth Sensitive, SECURITY.md, AGENTS.md ([#350](https://github.com/batonogov/terraform-provider-threexui/issues/350)) ([d900630](https://github.com/batonogov/terraform-provider-threexui/commit/d9006300d7718672d0bcdf8d5217066de5bd59d9))
* **blocker:** preserve WireGuard multi-client clients[] across the wire ([#342](https://github.com/batonogov/terraform-provider-threexui/issues/342)) ([#348](https://github.com/batonogov/terraform-provider-threexui/issues/348)) ([113c726](https://github.com/batonogov/terraform-provider-threexui/commit/113c726d246717f8210ad742ed42e385fe8a0943))

## [3.19.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.18.0...v3.19.0) (2026-07-13)


### Features

* add 3x-ui v3.5.0 support ([#332](https://github.com/batonogov/terraform-provider-threexui/issues/332)) ([5aa664c](https://github.com/batonogov/terraform-provider-threexui/commit/5aa664c42421ac15a939656757d6608949fe523c))

## [3.18.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.17.0...v3.18.0) (2026-06-30)


### Features

* support 3x-ui v3.4.2 (ldap, WireGuard multi-client, 2FA) ([#324](https://github.com/batonogov/terraform-provider-threexui/issues/324)) ([5ff2ff7](https://github.com/batonogov/terraform-provider-threexui/commit/5ff2ff7f0c72e065dd5171ae67e5e0db2229b74f))

## [3.17.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.16.0...v3.17.0) (2026-06-27)


### Features

* add threexui_node resource (Create/Read/Import) for cluster nodes ([#321](https://github.com/batonogov/terraform-provider-threexui/issues/321)) ([6be298c](https://github.com/batonogov/terraform-provider-threexui/commit/6be298ca5249f6b9d81d57f9d72bce17d454faef))
* add threexui_nodes data source (cluster node tree) ([#320](https://github.com/batonogov/terraform-provider-threexui/issues/320)) ([08325c1](https://github.com/batonogov/terraform-provider-threexui/commit/08325c189fd59b38c7cdcb99f687ee5dfdbd10b6))
* **node:** real Update + Delete for threexui_node (M3) ([#322](https://github.com/batonogov/terraform-provider-threexui/issues/322)) ([72b86a4](https://github.com/batonogov/terraform-provider-threexui/commit/72b86a441020012be97521afa19fd75a30bce423))
* **node:** write-only secrets for threexui_node (M4) ([#323](https://github.com/batonogov/terraform-provider-threexui/issues/323)) ([7d8d009](https://github.com/batonogov/terraform-provider-threexui/commit/7d8d009f76f7ae8c31f835ff94fcc7c71f7dabfc))


### Bug Fixes

* **test:** skip TestAccXrayVersionDrift before binary -timeout panic ([#312](https://github.com/batonogov/terraform-provider-threexui/issues/312)) ([15f1077](https://github.com/batonogov/terraform-provider-threexui/commit/15f10777e123ee5ac195d378e255d0673a4bf7b3))

## [3.16.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.15.0...v3.16.0) (2026-06-26)


### Features

* support 3x-ui v3.4.1 ([#310](https://github.com/batonogov/terraform-provider-threexui/issues/310)) ([0a405b2](https://github.com/batonogov/terraform-provider-threexui/commit/0a405b2398320fcd53015af6e58b7377acb9d458))

## [3.15.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.14.4...v3.15.0) (2026-06-25)


### Features

* add support for 3x-ui v3.4.0 ([#306](https://github.com/batonogov/terraform-provider-threexui/issues/306)) ([56a666d](https://github.com/batonogov/terraform-provider-threexui/commit/56a666d61fa12f986c12c5a5415d582a02963fe6))


### Bug Fixes

* **test:** bound TestAccXrayVersionDrift Step 2 budget and skip on pickup failure ([#308](https://github.com/batonogov/terraform-provider-threexui/issues/308)) ([7b23c2a](https://github.com/batonogov/terraform-provider-threexui/commit/7b23c2a3bdea0a39bf65a2921a7b7c3fd205d489))

## [3.14.4](https://github.com/batonogov/terraform-provider-threexui/compare/v3.14.3...v3.14.4) (2026-06-18)


### Bug Fixes

* **settings:** restart panel when subscription server settings change ([#303](https://github.com/batonogov/terraform-provider-threexui/issues/303)) ([9c3abe6](https://github.com/batonogov/terraform-provider-threexui/commit/9c3abe6840a1cca28870aa9de7346630e419057f))

## [3.14.3](https://github.com/batonogov/terraform-provider-threexui/compare/v3.14.2...v3.14.3) (2026-06-18)


### Bug Fixes

* keep client_id stable so metadata edits update in-place ([#298](https://github.com/batonogov/terraform-provider-threexui/issues/298)) ([d5012a0](https://github.com/batonogov/terraform-provider-threexui/commit/d5012a0ef89d4dff81fd7cc9ebdd562156b9084b))

## [3.14.2](https://github.com/batonogov/terraform-provider-threexui/compare/v3.14.1...v3.14.2) (2026-06-17)


### Bug Fixes

* **settings:** restart the panel when subscription server settings change ([#292](https://github.com/batonogov/terraform-provider-threexui/issues/292)) ([f1752d2](https://github.com/batonogov/terraform-provider-threexui/commit/f1752d28af4f1c143bc43ca47fe30c3ffab719e2))

## [3.14.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.14.0...v3.14.1) (2026-06-16)


### Bug Fixes

* add DeprecationMessage to panel_proxy schema ([#288](https://github.com/batonogov/terraform-provider-threexui/issues/288)) ([52249f8](https://github.com/batonogov/terraform-provider-threexui/commit/52249f84e67ffe47defec5b91f62e00dbfca6423))
* add DeprecationMessage to remaining deprecated attributes ([#290](https://github.com/batonogov/terraform-provider-threexui/issues/290)) ([f734c15](https://github.com/batonogov/terraform-provider-threexui/commit/f734c1562967785b62ecfae7e4f405da51170c45))

## [3.14.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.13.0...v3.14.0) (2026-06-15)


### Features

* support panel_outbound (outbound egress bridge) for 3x-ui v3.3.1+ ([#283](https://github.com/batonogov/terraform-provider-threexui/issues/283)) ([b9b8715](https://github.com/batonogov/terraform-provider-threexui/commit/b9b87153480763d10137e6dfcfedaa65e666d8f6))

## [3.13.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.12.0...v3.13.0) (2026-06-15)


### Features

* add 3x-ui v3.3.1 support ([#278](https://github.com/batonogov/terraform-provider-threexui/issues/278)) ([b9c47d8](https://github.com/batonogov/terraform-provider-threexui/commit/b9c47d8fef40443eba9511e8ba6823c78a630f05))
* add write-only secret arguments for Terraform 1.11+ ([#275](https://github.com/batonogov/terraform-provider-threexui/issues/275)) ([8eaa9e1](https://github.com/batonogov/terraform-provider-threexui/commit/8eaa9e10ae6990a64408c42de6dc7b1b8ed44c55))

## [3.12.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.11.1...v3.12.0) (2026-06-11)


### Features

* add acceptance tests with PostgreSQL backend ([#269](https://github.com/batonogov/terraform-provider-threexui/issues/269)) ([e7496c1](https://github.com/batonogov/terraform-provider-threexui/commit/e7496c194c316ba415a9363a929bb26c72975073))


### Bug Fixes

* make sub_id Optional+Computed for disaster recovery ([#272](https://github.com/batonogov/terraform-provider-threexui/issues/272)) ([8f74f01](https://github.com/batonogov/terraform-provider-threexui/commit/8f74f016496b7f9820df05d88cd10218a870c1f4))

## [3.11.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.11.0...v3.11.1) (2026-06-10)


### Bug Fixes

* use form-encoded probe for settings API and add 404 fallback ([#266](https://github.com/batonogov/terraform-provider-threexui/issues/266)) ([3108990](https://github.com/batonogov/terraform-provider-threexui/commit/31089903479f1c7decc737a13b5807d233b2f4f9))

## [3.11.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.10.0...v3.11.0) (2026-06-09)


### Features

* add 3x-ui v3.3.0 support, drop v2.9.x and v3.0.x ([#264](https://github.com/batonogov/terraform-provider-threexui/issues/264)) ([2d4a6f0](https://github.com/batonogov/terraform-provider-threexui/commit/2d4a6f039fcdc87c0d248167fdcddb928306a831))

## [3.10.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.9.0...v3.10.0) (2026-06-08)


### Features

* add schema validators for provider config and core resources ([#258](https://github.com/batonogov/terraform-provider-threexui/issues/258)) ([5543e7c](https://github.com/batonogov/terraform-provider-threexui/commit/5543e7cc8a07117be401d9c5cd4d5fdf84b8d3bf))


### Bug Fixes

* increase waitForXrayVersion poll budget and retry install on stale version ([#263](https://github.com/batonogov/terraform-provider-threexui/issues/263)) ([fbd4480](https://github.com/batonogov/terraform-provider-threexui/commit/fbd448049c47d5f6ae5fc870353fe9c629d8c1ce)), closes [#262](https://github.com/batonogov/terraform-provider-threexui/issues/262)

## [3.9.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.8.0...v3.9.0) (2026-06-07)


### Features

* add restart_xray attribute to inbound and inbound_client resources ([#244](https://github.com/batonogov/terraform-provider-threexui/issues/244)) ([08adeab](https://github.com/batonogov/terraform-provider-threexui/commit/08adeab62ddd8e98f58f33c1901b61d76cf86906)), closes [#214](https://github.com/batonogov/terraform-provider-threexui/issues/214)
* implement import support for panel_user resource ([6d9ba23](https://github.com/batonogov/terraform-provider-threexui/commit/6d9ba23a1a9563ef85a6271bacdcede8e5d75cdd))
* implement import support for panel_user resource ([5b819fb](https://github.com/batonogov/terraform-provider-threexui/commit/5b819fb4f4a50b78fde944abc6cae8cc9ffafec2)), closes [#247](https://github.com/batonogov/terraform-provider-threexui/issues/247)
* support THREEXUI_* environment variables in provider configuration ([#255](https://github.com/batonogov/terraform-provider-threexui/issues/255)) ([0b19a68](https://github.com/batonogov/terraform-provider-threexui/commit/0b19a68d47b2668fcdc4cdbce82265eb010b00ee)), closes [#249](https://github.com/batonogov/terraform-provider-threexui/issues/249)

## [3.8.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.7.1...v3.8.0) (2026-06-06)


### Features

* add 3x-ui v3.2.7 & v3.2.8 compatibility ([#241](https://github.com/batonogov/terraform-provider-threexui/issues/241)) ([7b1ab3c](https://github.com/batonogov/terraform-provider-threexui/commit/7b1ab3c9b0712235d6eff13ab4247a956f95a3d8))
* add metrics block to xray_basics resource ([#243](https://github.com/batonogov/terraform-provider-threexui/issues/243)) ([cc00334](https://github.com/batonogov/terraform-provider-threexui/commit/cc00334ea42c5b88b961458ce59adc6507c03e87)), closes [#220](https://github.com/batonogov/terraform-provider-threexui/issues/220)

## [3.7.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.7.0...v3.7.1) (2026-06-05)


### Bug Fixes

* use Reality stream settings for VLESS client tests with flow ([#238](https://github.com/batonogov/terraform-provider-threexui/issues/238)) ([2b68ad4](https://github.com/batonogov/terraform-provider-threexui/commit/2b68ad44a61e5cb809a268db422ccd6e231dd7d8))

## [3.7.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.6.4...v3.7.0) (2026-06-01)


### Features

* 3x-ui v3.2.0 compatibility ([#232](https://github.com/batonogov/terraform-provider-threexui/issues/232)) ([7b5c99c](https://github.com/batonogov/terraform-provider-threexui/commit/7b5c99cbf938de21bf4e6f9414db83c8e538ca3a))


### Bug Fixes

* quarantine v3.0.0/v3.0.1 in TestAccXrayVersionDrift ([#224](https://github.com/batonogov/terraform-provider-threexui/issues/224)) ([#229](https://github.com/batonogov/terraform-provider-threexui/issues/229)) ([9514532](https://github.com/batonogov/terraform-provider-threexui/commit/951453269eae7a1ef529c417074b953d03a5b93e))
* skip socks/dokodemo-door tests on 3x-ui v3.2.0+ ([#235](https://github.com/batonogov/terraform-provider-threexui/issues/235)) ([b173a12](https://github.com/batonogov/terraform-provider-threexui/commit/b173a12240b4935e233b7f2bac481064b3d50488))

## [3.6.4](https://github.com/batonogov/terraform-provider-threexui/compare/v3.6.3...v3.6.4) (2026-05-27)


### Bug Fixes

* (known after apply) noise in xray_routing and xray_outbounds ([#228](https://github.com/batonogov/terraform-provider-threexui/issues/228)) ([b33c4f1](https://github.com/batonogov/terraform-provider-threexui/commit/b33c4f14987cc33d5a600bc4e0cc9829e3cb9e5c))
* blackhole_settings.response_type false diff on import ([#226](https://github.com/batonogov/terraform-provider-threexui/issues/226)) ([2e84dd0](https://github.com/batonogov/terraform-provider-threexui/commit/2e84dd0b30670ad4734decc4b093c8c70ee623f3)), closes [#218](https://github.com/batonogov/terraform-provider-threexui/issues/218)
* reject API routing rules in threexui_xray_routing ([#221](https://github.com/batonogov/terraform-provider-threexui/issues/221)) ([8c88334](https://github.com/batonogov/terraform-provider-threexui/commit/8c88334f4f95afd8457ea9ee596beb8dc78b7ced))
* retry transient errors in read-after-write for inbound/inbound_client ([#227](https://github.com/batonogov/terraform-provider-threexui/issues/227)) ([c95f7a1](https://github.com/batonogov/terraform-provider-threexui/commit/c95f7a1b96b4143e06e845c0bddab42166ad1c33))

## [3.6.3](https://github.com/batonogov/terraform-provider-threexui/compare/v3.6.2...v3.6.3) (2026-05-27)


### Bug Fixes

* prevent inconsistent result when xray_basics optional blocks omitted ([#212](https://github.com/batonogov/terraform-provider-threexui/issues/212)) ([a947e16](https://github.com/batonogov/terraform-provider-threexui/commit/a947e164ca0fc02fe7ed4d8e6b0a89122a79be3f)), closes [#208](https://github.com/batonogov/terraform-provider-threexui/issues/208)

## [3.6.2](https://github.com/batonogov/terraform-provider-threexui/compare/v3.6.1...v3.6.2) (2026-05-26)


### Bug Fixes

* preserve empty tcp_settings block through expand/flatten cycle ([#210](https://github.com/batonogov/terraform-provider-threexui/issues/210)) ([3bd3328](https://github.com/batonogov/terraform-provider-threexui/commit/3bd332897a78f15e3c33608a722edb9de31de1d5))

## [3.6.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.6.0...v3.6.1) (2026-05-26)


### Bug Fixes

* prevent perpetual plan drift from API routing rule ([#204](https://github.com/batonogov/terraform-provider-threexui/issues/204)) ([8803665](https://github.com/batonogov/terraform-provider-threexui/commit/88036657117c59044d14422fda8874e0a101903e))

## [3.6.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.5.0...v3.6.0) (2026-05-25)


### Features

* add 3x-ui v3.1.0 support ([#201](https://github.com/batonogov/terraform-provider-threexui/issues/201)) ([3ecd8c0](https://github.com/batonogov/terraform-provider-threexui/commit/3ecd8c0cd624057b1bca6ae8c8e50dab849532c1))
* add govulncheck to CI and enable GitHub Sponsors ([#192](https://github.com/batonogov/terraform-provider-threexui/issues/192)) ([6874072](https://github.com/batonogov/terraform-provider-threexui/commit/687407213925b4e837a3e140bc098894ce810efd))
* enable gosec linter and capture container logs on CI failure ([#194](https://github.com/batonogov/terraform-provider-threexui/issues/194)) ([500731b](https://github.com/batonogov/terraform-provider-threexui/commit/500731b07d18ac2cd0f47f70801a8856f3d3c4fe))
* report test coverage to Codecov and add badge ([#193](https://github.com/batonogov/terraform-provider-threexui/issues/193)) ([fd286c3](https://github.com/batonogov/terraform-provider-threexui/commit/fd286c3e3440cc933317b6e0e0ce10d50b69b31b))


### Bug Fixes

* stabilize CI compat matrix around Xray version rate limits ([#189](https://github.com/batonogov/terraform-provider-threexui/issues/189)) ([9515338](https://github.com/batonogov/terraform-provider-threexui/commit/9515338fa16897ff36595d93c641df4e10eff387))
* suppress drift on traffic counter fields during inbound update ([#203](https://github.com/batonogov/terraform-provider-threexui/issues/203)) ([9b12113](https://github.com/batonogov/terraform-provider-threexui/commit/9b121138681ab119bcda23a61abd61b6dc8e85f7))

## [3.5.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.4.0...v3.5.0) (2026-05-15)


### Features

* add bootstrap provider credentials ([#187](https://github.com/batonogov/terraform-provider-threexui/issues/187)) ([846fcde](https://github.com/batonogov/terraform-provider-threexui/commit/846fcdec9d8f3d76082209f8ff56172911d87d84))

## [3.4.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.3.0...v3.4.0) (2026-05-14)


### Features

* support 3x-ui v3.0.1 ([#180](https://github.com/batonogov/terraform-provider-threexui/issues/180)) ([15d5766](https://github.com/batonogov/terraform-provider-threexui/commit/15d5766b48bb0d54e61cdaf22b605fc2fd6815d0))
* support 3x-ui v3.0.2 ([#185](https://github.com/batonogov/terraform-provider-threexui/issues/185)) ([05fc657](https://github.com/batonogov/terraform-provider-threexui/commit/05fc65718edd7d76e185305869884e89d5bea770))


### Bug Fixes

* preserve redacted panel setting secrets ([#186](https://github.com/batonogov/terraform-provider-threexui/issues/186)) ([9562079](https://github.com/batonogov/terraform-provider-threexui/commit/95620799638ebeeffa7b04363a88fc5126d5cfda))

## [3.3.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.2.0...v3.3.0) (2026-05-13)


### Features

* support 3x-ui v3.0.0 ([#177](https://github.com/batonogov/terraform-provider-threexui/issues/177)) ([07f155e](https://github.com/batonogov/terraform-provider-threexui/commit/07f155e18071cb509ca2b816a1ac02fc71cddf2e))

## [3.2.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.1.1...v3.2.0) (2026-05-10)


### Features

* add 3x-ui v2.9.4 support ([#173](https://github.com/batonogov/terraform-provider-threexui/issues/173)) ([f86c9b7](https://github.com/batonogov/terraform-provider-threexui/commit/f86c9b7559189f859e3d10e7b13a355973001a02))

## [3.1.1](https://github.com/batonogov/terraform-provider-threexui/compare/v3.1.0...v3.1.1) (2026-04-28)


### Bug Fixes

* retry post-write reads to absorb 3x-ui SQLite visibility lag ([#158](https://github.com/batonogov/terraform-provider-threexui/issues/158)) ([7ee68fb](https://github.com/batonogov/terraform-provider-threexui/commit/7ee68fbeccc43b8e54cf2c20426ac7b5ef5a12d7))
* stabilize CI flakes (DeleteInbound retry-with-verify, readiness gate, per-job retry) ([#162](https://github.com/batonogov/terraform-provider-threexui/issues/162)) ([afa1a7e](https://github.com/batonogov/terraform-provider-threexui/commit/afa1a7e083f79a2445944c789e3fa67a152e93b9))

## [3.1.0](https://github.com/batonogov/terraform-provider-threexui/compare/v3.0.0...v3.1.0) (2026-04-27)


### Features

* support 3x-ui v2.9.3 and hysteria2 protocol alias ([#155](https://github.com/batonogov/terraform-provider-threexui/issues/155)) ([2c099cb](https://github.com/batonogov/terraform-provider-threexui/commit/2c099cbf31e26c10ef0ef99f83977e0fe1272cd8))

## [3.0.0](https://github.com/batonogov/terraform-provider-threexui/compare/v2.1.0...v3.0.0) (2026-04-27)


### ⚠ BREAKING CHANGES

* The `json` attribute of the `threexui_xray_config` data source is now marked Sensitive. Existing `output` blocks that reference `data.threexui_xray_config.<name>.json` (or values derived from it) must add `sensitive = true`, or wrap safe fields in `nonsensitive(...)`. Otherwise `terraform plan` fails with "Output refers to sensitive values".
* The `inbounds` attribute of the `threexui_inbounds` data source is now marked Sensitive. Existing `output` blocks that reference `data.threexui_inbounds.<name>.inbounds` (or values derived from it) must add `sensitive = true`, or wrap safe fields in `nonsensitive(...)`. Otherwise `terraform plan` fails with "Output refers to sensitive values".
* The `json` attribute of the `threexui_settings` data source was marked Sensitive in #143. Existing `output` blocks that reference it must add `sensitive = true`, otherwise `terraform plan` fails with "Output refers to sensitive values". The footer is recorded here so release-please surfaces the user-visible impact in the next release notes (the original PR landed without it).

### Bug Fixes

* confirm inbound deletion via poll-and-retry ([#136](https://github.com/batonogov/terraform-provider-threexui/issues/136)) ([#141](https://github.com/batonogov/terraform-provider-threexui/issues/141)) ([8c6fd4e](https://github.com/batonogov/terraform-provider-threexui/commit/8c6fd4ef03aed7452463f22a83c402ae9cb4a817))
* mark threexui_inbounds data source 'inbounds' attribute as sensitive ([#145](https://github.com/batonogov/terraform-provider-threexui/issues/145)) ([1ee5993](https://github.com/batonogov/terraform-provider-threexui/commit/1ee59938d77898de6c9b1afda353abfe4b7b2265)), closes [#138](https://github.com/batonogov/terraform-provider-threexui/issues/138)
* mark threexui_settings data source 'json' attribute as sensitive ([#143](https://github.com/batonogov/terraform-provider-threexui/issues/143)) ([d2bcee1](https://github.com/batonogov/terraform-provider-threexui/commit/d2bcee1661239ef4026b18159b356f38a4d30e03)), closes [#137](https://github.com/batonogov/terraform-provider-threexui/issues/137)
* mark threexui_xray_config data source 'json' attribute as sensitive ([#146](https://github.com/batonogov/terraform-provider-threexui/issues/146)) ([15b1ec7](https://github.com/batonogov/terraform-provider-threexui/commit/15b1ec74a257bf1154c371a52ad97d8c2ba216d1)), closes [#139](https://github.com/batonogov/terraform-provider-threexui/issues/139)
* re-read inbound after create/update to prevent flaky tests ([fa4b609](https://github.com/batonogov/terraform-provider-threexui/commit/fa4b6096d211913cd563aa68df1f316ca1fa4479))
* re-read inbound after create/update to prevent flaky tests ([#131](https://github.com/batonogov/terraform-provider-threexui/issues/131)) ([51d9d0c](https://github.com/batonogov/terraform-provider-threexui/commit/51d9d0c7616b4ca3cbfbed524e3f428a62b93cd4))
* retry transient 5xx on write endpoints ([#134](https://github.com/batonogov/terraform-provider-threexui/issues/134)) ([#142](https://github.com/batonogov/terraform-provider-threexui/issues/142)) ([71f66cc](https://github.com/batonogov/terraform-provider-threexui/commit/71f66cc998aa5a3c6163b413f2ba60d2e4f9d3b8))


### Documentation

* warn about sensitive output requirement for threexui_settings.json ([#144](https://github.com/batonogov/terraform-provider-threexui/issues/144)) ([c5927be](https://github.com/batonogov/terraform-provider-threexui/commit/c5927be49422da29c7cfb6bb217a1166cf23b7c0)), closes [#137](https://github.com/batonogov/terraform-provider-threexui/issues/137)

## [2.1.0](https://github.com/batonogov/terraform-provider-threexui/compare/v2.0.0...v2.1.0) (2026-04-24)


### Features

* add enable_parallel_query and use_system_hosts to threexui_xray_dns ([5d2d482](https://github.com/batonogov/terraform-provider-threexui/commit/5d2d4824a909cce0f486bfd0509245fef98aac57)), closes [#65](https://github.com/batonogov/terraform-provider-threexui/issues/65)
* add enable_parallel_query and use_system_hosts to xray_dns ([74886ec](https://github.com/batonogov/terraform-provider-threexui/commit/74886ec8c432f3b98f1f9b1ca851e7c8dfae44c9))
* add mixed_settings block for mixed inbound protocol ([2058039](https://github.com/batonogov/terraform-provider-threexui/commit/20580390c1330d7ee6bd1be9c4946537b4668bf2))
* add mixed_settings block for mixed inbound protocol ([8872854](https://github.com/batonogov/terraform-provider-threexui/commit/8872854d6959f0c59ed26ac14257131b7234da36)), closes [#64](https://github.com/batonogov/terraform-provider-threexui/issues/64)
* add xPadding fields to xhttp_settings block ([5e3ae5c](https://github.com/batonogov/terraform-provider-threexui/commit/5e3ae5c1ae6dc13e546635dcfae1357eb77b0042))
* add xPadding fields to xhttp_settings block ([846d99d](https://github.com/batonogov/terraform-provider-threexui/commit/846d99d2de0d5b37c634bce1d6d99feef038ec84)), closes [#122](https://github.com/batonogov/terraform-provider-threexui/issues/122)
* expose VLESS vision testseed as first-class field ([8ee3156](https://github.com/batonogov/terraform-provider-threexui/commit/8ee31567c53f8e54c40338a82cc18efb909df98c))
* replace curated compat test list with version-aware skipping ([32d80d0](https://github.com/batonogov/terraform-provider-threexui/commit/32d80d035d44aa60e9c8fff01c1157199f673a21))
* version-aware test skipping for compat matrix ([3198c0e](https://github.com/batonogov/terraform-provider-threexui/commit/3198c0ef2a1a5f513008573946780b6747bd856d))


### Bug Fixes

* add supported versions to SECURITY.md and docs link to issue config ([7555da5](https://github.com/batonogov/terraform-provider-threexui/commit/7555da59f491f4fe505e2cc484b15cbb8d374c37))
* add version constraint and parameterize insecure_skip_verify in multi-server example ([83650db](https://github.com/batonogov/terraform-provider-threexui/commit/83650db6cfc15c96791a3d70ffb7112a9632eae4))
* apply terraform fmt to trojan example ([11b41d3](https://github.com/batonogov/terraform-provider-threexui/commit/11b41d313ae041fe288973e97e600051c5b27774))
* correct VLESS Reality link, add version constraints and TLS warning ([b615c3e](https://github.com/batonogov/terraform-provider-threexui/commit/b615c3ef5445874b5666b1b604cf00020a1197de))
* **docs:** remove duplicate VLESS Reality row from Examples table ([83a50a7](https://github.com/batonogov/terraform-provider-threexui/commit/83a50a7508722e6969a20db1244354b45e3492d1))
* document sub_json_fragment/sub_json_noises format change for 3x-ui v2.9.2 ([88bb0d8](https://github.com/batonogov/terraform-provider-threexui/commit/88bb0d8b8f9e9d901ffe2096040a7b3282b5673f)), closes [#121](https://github.com/batonogov/terraform-provider-threexui/issues/121)
* document sub_json_fragment/sub_json_noises format change for v2.9.2 ([cb8cb59](https://github.com/batonogov/terraform-provider-threexui/commit/cb8cb593b8c50e2a17b4abd68e4c38b18f60296d))
* poll for xray version after async InstallXray in provider ([50032fc](https://github.com/batonogov/terraform-provider-threexui/commit/50032fc6cabdfcb1a1c14f37456ac9d262041817))
* remove incompatible ConfigPlanChecks.PreApply from PlanOnly step ([3f375f9](https://github.com/batonogov/terraform-provider-threexui/commit/3f375f99c571414c1c4e244939dd844927348b2e))
* remove incompatible ConfigPlanChecks.PreApply from PlanOnly step ([0aed523](https://github.com/batonogov/terraform-provider-threexui/commit/0aed5233a13af90d2fc5f7f66b960322e10f512a)), closes [#126](https://github.com/batonogov/terraform-provider-threexui/issues/126)
* restore waitForXrayVersion with corrected comment ([dbfcc36](https://github.com/batonogov/terraform-provider-threexui/commit/dbfcc365229c4ad9f8305675e6df76ea9c21afc1))
* revert unnecessary polling in xray_version, fix version annotations ([e8a53ad](https://github.com/batonogov/terraform-provider-threexui/commit/e8a53ada31d85f7b5a1ccdc8d7a045d37b9b2bf8))
* revert unnecessary polling in xray_version, fix version annotations ([921d9b0](https://github.com/batonogov/terraform-provider-threexui/commit/921d9b03164a44fe9ca7dbc99c5b9a5994b13b3f))
* skip panel_general tests on 3x-ui &lt; v2.8.10 and fix XrayVersionDrift test ([e17d3d2](https://github.com/batonogov/terraform-provider-threexui/commit/e17d3d251574e306e813d6c328eaa98a0ae85079))
* trigger CI on CHANGELOG.md changes for release PRs ([ac6f448](https://github.com/batonogov/terraform-provider-threexui/commit/ac6f448b182d50f476f03ef7b8f4856cc6779297))
* trigger CI on CHANGELOG.md changes for release PRs ([131d8ca](https://github.com/batonogov/terraform-provider-threexui/commit/131d8ca0ff7f86bbc409cdbe81fc026c1e2a10be))
* wait for async InstallXray to complete before asserting drift ([d53bb7d](https://github.com/batonogov/terraform-provider-threexui/commit/d53bb7db26277aa638743260f391a7fa43416858))

## [2.0.0](https://github.com/batonogov/terraform-provider-threexui/compare/v1.0.0...v2.0.0) (2026-04-24)


### ⚠ BREAKING CHANGES

* sub_id is now read-only; remove any sub_id assignments from inbound_client configs.

### Bug Fixes

* add UseStateForUnknown to inbound_client, make sub_id Computed-only ([#104](https://github.com/batonogov/terraform-provider-threexui/issues/104)) ([078ec72](https://github.com/batonogov/terraform-provider-threexui/commit/078ec72b35259ed70145645ad891fa5bf794f623))

## [1.0.0](https://github.com/batonogov/terraform-provider-threexui/compare/v0.6.1...v1.0.0) (2026-04-24)


### ⚠ BREAKING CHANGES

* users who explicitly specify reality_settings.settings must change from block syntax (settings { ... }) to attribute syntax (settings = { ... }). Most users omit this block entirely and are unaffected.

### Bug Fixes

* convert reality_settings.settings from block to attribute ([#102](https://github.com/batonogov/terraform-provider-threexui/issues/102)) ([a89e7ea](https://github.com/batonogov/terraform-provider-threexui/commit/a89e7eaa458edb5706ff7c2a99308c413b1f98d5))

## [0.6.1](https://github.com/batonogov/terraform-provider-threexui/compare/v0.6.0...v0.6.1) (2026-04-24)


### Bug Fixes

* eliminate false drift on import for Optional+Computed inbound fields ([#100](https://github.com/batonogov/terraform-provider-threexui/issues/100)) ([2fcc66f](https://github.com/batonogov/terraform-provider-threexui/commit/2fcc66fa29fa311e16e5a8111158042ba7b54381))

## [0.6.0](https://github.com/batonogov/terraform-provider-threexui/compare/v0.5.3...v0.6.0) (2026-04-21)


### Features

* support 3x-ui 2.9.0 ([#91](https://github.com/batonogov/terraform-provider-threexui/issues/91)) ([db546eb](https://github.com/batonogov/terraform-provider-threexui/commit/db546eb5e160e6db428a52a551dbdc4c0204ed9f))

## [0.5.3](https://github.com/batonogov/terraform-provider-threexui/compare/v0.5.2...v0.5.3) (2026-03-20)


### Bug Fixes

* add markdownlint to pre-commit and fix MD060 violations ([#89](https://github.com/batonogov/terraform-provider-threexui/issues/89)) ([5fd6f6d](https://github.com/batonogov/terraform-provider-threexui/commit/5fd6f6d897dc6bbdc148578f25162b71dafb21a8))
* align tunnel inbound support with 3x-ui 2.8.11 ([#87](https://github.com/batonogov/terraform-provider-threexui/issues/87)) ([29adfce](https://github.com/batonogov/terraform-provider-threexui/commit/29adfced9a0138de42627b2f4132c213701fd584))

## [0.5.2](https://github.com/batonogov/terraform-provider-threexui/compare/v0.5.1...v0.5.2) (2026-03-14)


### Bug Fixes

* update client base_path after web_base_path change ([#81](https://github.com/batonogov/terraform-provider-threexui/issues/81)) ([2e890b0](https://github.com/batonogov/terraform-provider-threexui/commit/2e890b0ee3940765eade887ebd98a85a5f22182e))

## [0.5.1](https://github.com/batonogov/terraform-provider-threexui/compare/v0.5.0...v0.5.1) (2026-03-12)


### Bug Fixes

* fail fast when inbound settings JSON cannot be parsed ([#61](https://github.com/batonogov/terraform-provider-threexui/issues/61)) ([b764019](https://github.com/batonogov/terraform-provider-threexui/commit/b764019f9a85f75aa783e6308980d3a40ec6cefe))

## [0.5.0](https://github.com/batonogov/terraform-provider-threexui/compare/v0.4.2...v0.5.0) (2026-03-12)


### Features

* add threexui_client_traffics data source ([#56](https://github.com/batonogov/terraform-provider-threexui/issues/56)) ([4bb4d78](https://github.com/batonogov/terraform-provider-threexui/commit/4bb4d785584cbb682dc8926715cc7c0958c16f85))
* add threexui_xray_version resource ([#57](https://github.com/batonogov/terraform-provider-threexui/issues/57)) ([1dd9be6](https://github.com/batonogov/terraform-provider-threexui/commit/1dd9be6c8f217d01039094069859ec6cbb55c355))


### Bug Fixes

* normalize inbound listen and total defaults ([#53](https://github.com/batonogov/terraform-provider-threexui/issues/53)) ([f6e700e](https://github.com/batonogov/terraform-provider-threexui/commit/f6e700ec4d42d7c11e57eb31554a614310e9b2de))
* serialize panel_general updates with settingsMu and xrayTemplateMu ([#59](https://github.com/batonogov/terraform-provider-threexui/issues/59)) ([951b4cc](https://github.com/batonogov/terraform-provider-threexui/commit/951b4cc9c6e626dd12e7f96c6bb6c3c33b46b5d8))

## [0.4.2](https://github.com/batonogov/terraform-provider-threexui/compare/v0.4.1...v0.4.2) (2026-03-06)


### Bug Fixes

* remove email fallback in getClientIDFromModel ([#46](https://github.com/batonogov/terraform-provider-threexui/issues/46)) ([d496d93](https://github.com/batonogov/terraform-provider-threexui/commit/d496d9368bf30611b67bc284f0d9faafffcacc46))

## [0.4.1](https://github.com/batonogov/terraform-provider-threexui/compare/v0.4.0...v0.4.1) (2026-03-06)


### Bug Fixes

* add default empty string for inbound remark attribute ([#44](https://github.com/batonogov/terraform-provider-threexui/issues/44)) ([64cd3e3](https://github.com/batonogov/terraform-provider-threexui/commit/64cd3e300b0d41bd5660f03b75e8e6b28a7d84d9))

## [0.4.0](https://github.com/batonogov/terraform-provider-threexui/compare/v0.3.0...v0.4.0) (2026-03-06)


### Features

* add 3x-ui v2.8.10 support and xray_outbound_test_url attribute ([#24](https://github.com/batonogov/terraform-provider-threexui/issues/24)) ([4d624b2](https://github.com/batonogov/terraform-provider-threexui/commit/4d624b2f72f5b9a23e5c35c0c628c15dfe41b5df))
* add 3x-ui v2.8.11 support ([#34](https://github.com/batonogov/terraform-provider-threexui/issues/34)) ([36556eb](https://github.com/batonogov/terraform-provider-threexui/commit/36556eb4e44cb63ec498fd3ac7a58b7a7dfaf41f))
* add inbound tests and uuid helpers ([09fd75a](https://github.com/batonogov/terraform-provider-threexui/commit/09fd75ae42719ed86479b35e01c78d1de1417718))
* add inbound tests and uuid helpers ([e625755](https://github.com/batonogov/terraform-provider-threexui/commit/e6257552cb437a712e287cbf42a15eff47066749))
* add pre-commit hooks configuration ([99d5056](https://github.com/batonogov/terraform-provider-threexui/commit/99d5056724fc72ea24f7617f4716740e5cf64309))
* add pre-commit hooks configuration ([414aff5](https://github.com/batonogov/terraform-provider-threexui/commit/414aff52b837e9b64bdb876c3bf083d4b8e49f6b))
* add pre-commit hooks configuration ([433393e](https://github.com/batonogov/terraform-provider-threexui/commit/433393e83ad25d3a9c556231e46295c970c4387c))
* add provider settings resources and update examples ([705483f](https://github.com/batonogov/terraform-provider-threexui/commit/705483f7f37c60ee21dcfe87d07c615e68573aa3))
* add provider settings resources and update examples ([ba8fea7](https://github.com/batonogov/terraform-provider-threexui/commit/ba8fea7abed84b0ac3731e863ac7ac05b4fa835d))
* add Release Please for automated releases ([#29](https://github.com/batonogov/terraform-provider-threexui/issues/29)) ([02f4880](https://github.com/batonogov/terraform-provider-threexui/commit/02f48801e6204c71d5d584ac039806d746dac52d))
* add test plans, update CLAUDE.md and provider improvements ([ec6fcb2](https://github.com/batonogov/terraform-provider-threexui/commit/ec6fcb2003c1b2373cbb20b2469200664404849e))
* add threexui_online_clients data source ([#40](https://github.com/batonogov/terraform-provider-threexui/issues/40)) ([d7df4d2](https://github.com/batonogov/terraform-provider-threexui/commit/d7df4d2cf012c20968a97f689a4b202da9fc684a))
* add threexui_panel_user resource ([#18](https://github.com/batonogov/terraform-provider-threexui/issues/18)) ([e166e92](https://github.com/batonogov/terraform-provider-threexui/commit/e166e92a2601ce25bce44abf2199e92e07903317))
* implement initial provider skeleton ([2877b0f](https://github.com/batonogov/terraform-provider-threexui/commit/2877b0f7e5f4dfc4df303cee104e000bcb6d1c75))
* implement initial provider skeleton ([5a590f6](https://github.com/batonogov/terraform-provider-threexui/commit/5a590f628c4bb8bfb65666e061c0bf91e271a0bb))
* replace JSON string attributes with typed blocks in threexui_inbound ([#22](https://github.com/batonogov/terraform-provider-threexui/issues/22)) ([1a979a7](https://github.com/batonogov/terraform-provider-threexui/commit/1a979a7f6c624ef19c6292aab727ebf0fdbd5f37))
* switch from OpenTofu to Terraform Registry ([#28](https://github.com/batonogov/terraform-provider-threexui/issues/28)) ([fb26e72](https://github.com/batonogov/terraform-provider-threexui/commit/fb26e72d47deab62b5d8b1346bd5ed5b1320ffaf))


### Bug Fixes

* add mutex and subset diff suppress for xray settings ([906c4a7](https://github.com/batonogov/terraform-provider-threexui/commit/906c4a709474ac7156af544e20ef7cc4690dd494))
* add warnings for 2FA/base_path and fix sub_json_enable persistence ([0cf40e2](https://github.com/batonogov/terraform-provider-threexui/commit/0cf40e22912a5bb4e7e235dac58c2258288349b3))
* **ci:** use golangci-lint-action v7 for golangci-lint v2 config ([418de60](https://github.com/batonogov/terraform-provider-threexui/commit/418de60fc01139186e6014ff260f1a5f3386e587))
* correct TestBuildAndFlattenSettings assertions ([4825e90](https://github.com/batonogov/terraform-provider-threexui/commit/4825e90bada76aa587920b11f584af5ee75ccea9))
* fix remaining golangci-lint v2 config issues ([f1b10c4](https://github.com/batonogov/terraform-provider-threexui/commit/f1b10c405080b77076cbcb0e29418e414e87ca7f))
* make client email required to prevent 3x-ui SQL errors ([4bc4543](https://github.com/batonogov/terraform-provider-threexui/commit/4bc454376dad4f02a87521f388bd6e72ef52803c))
* migrate .golangci.yml to v2 format ([e57f5b7](https://github.com/batonogov/terraform-provider-threexui/commit/e57f5b73d0d16b9089b2fa5b22e3ba48144b00a4))
* preserve testseed in state by adding flattenIntList to flattenSettings ([c6e6c59](https://github.com/batonogov/terraform-provider-threexui/commit/c6e6c592c26283cbca92ec844bef3922788d1821))
* remove unsupported max-issues options from golangci-lint v2 config ([b471654](https://github.com/batonogov/terraform-provider-threexui/commit/b4716543cf10436ac715f539d9c1152ddc6a9445))
* reset CHANGELOG.md so Release Please recreates v0.3.0 ([#39](https://github.com/batonogov/terraform-provider-threexui/issues/39)) ([9207002](https://github.com/batonogov/terraform-provider-threexui/commit/920700217cf1cecded95070bfb35636c45f257a8))
* resolve all golangci-lint v2 warnings ([45015cc](https://github.com/batonogov/terraform-provider-threexui/commit/45015ccfaf315509cbddaebe53afac95ba64dd8f))
* run GoReleaser inside release-please workflow ([#35](https://github.com/batonogov/terraform-provider-threexui/issues/35)) ([c7599b4](https://github.com/batonogov/terraform-provider-threexui/commit/c7599b4eca9515783f978cf6852596620096458d))

## Changelog
