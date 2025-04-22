module github.com/uoregon-libraries/newspaper-curation-app

go 1.23.0

toolchain go1.24.2

require (
	github.com/Nerdmaster/magicsql v0.11.0
	github.com/go-sql-driver/mysql v1.7.1
	github.com/google/go-cmp v0.6.0
	github.com/gorilla/mux v1.7.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/pressly/goose/v3 v3.13.4
	github.com/tidwall/gjson v1.17.1
	github.com/tidwall/sjson v1.2.5
	github.com/uoregon-libraries/gopkg v0.30.2
	golang.org/x/crypto v0.31.0
	golang.org/x/text v0.23.0
)

require (
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/chavacava/garif v0.1.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/mgechev/dots v0.0.0-20210922191527-e955255bf517 // indirect
	github.com/mgechev/revive v1.8.0 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/telemetry v0.0.0-20240522233618-39ace7a40ae7 // indirect
	golang.org/x/tools v0.31.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
)

tool (
	github.com/mgechev/revive
	golang.org/x/tools/cmd/goimports
	golang.org/x/vuln/cmd/govulncheck
)
