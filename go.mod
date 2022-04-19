module github.com/solarlune/masterplan

go 1.13

require (
	github.com/adrg/xdg v0.4.0
	github.com/atotto/clipboard v0.1.4
	github.com/blang/semver v3.5.1+incompatible
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/chonla/roman-number-go v0.0.0-20181101035413-6768129de021
	github.com/gabriel-vasile/mimetype v1.4.0
	github.com/gen2brain/raylib-go v0.0.0-20210623105341-8ff192f923a5
	github.com/goware/urlx v0.3.1
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/ncruces/zenity v0.8.2
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/tanema/gween v0.0.0-20220318192052-2db1c2d931bd
	github.com/tidwall/gjson v1.14.0
	github.com/tidwall/sjson v1.2.4
)

// The below line replaces the normal raylib-go dependency with my branch that has the config.h tweaked to
// remove screenshot-taking because we're do it manually in MasterPlan.
replace github.com/gen2brain/raylib-go => github.com/solarlune/raylib-go v0.0.0-20210122080031-04529085ce96
