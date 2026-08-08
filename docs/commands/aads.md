<!-- Generated from the aads command set. -->

# `aads` command reference

`aads` calls the Apple Ads Campaign Management API 5.

```text
aads <command> <action> [flags]
```

## Global options

| Option | Description |
|---|---|
| `--output <json\|table\|yaml>`, `-o` | Choose the output format. The default is JSON. |
| `--org-id <id>` | Override the configured organization ID. |
| `--fields <names>` | Request a comma-separated set of fields where Apple supports partial fetch. |
| `--currency <code>` | Override the currency used for money values built from flags. |
| `--envelope` | Include Apple pagination, metadata, and totals with the response data. |
| `--verbose`, `-v` | Print request and retry details to standard error. |
| `--help`, `-h` | Show help. |
| `--version` | Print the version. |

Commands that create, update, or delete resources accept `--dry-run`. Delete and bulk-delete commands accept `--yes`.

## Command tree

```text
aads
├── configure
├── version
├── acls
│   ├── list
│   └── me
├── campaigns
│   ├── create
│   ├── get
│   ├── list
│   ├── find
│   ├── update
│   └── delete
├── budgetorders
│   ├── create
│   ├── get
│   ├── list
│   └── update
├── adgroups
│   ├── create
│   ├── get
│   ├── list
│   ├── find
│   ├── find-all
│   ├── update
│   └── delete
├── keywords
│   ├── create
│   ├── get
│   ├── list
│   ├── find
│   ├── find-campaign
│   ├── update
│   ├── delete
│   └── delete-one
├── negatives
│   ├── campaign-create
│   ├── campaign-get
│   ├── campaign-list
│   ├── campaign-find
│   ├── campaign-update
│   ├── campaign-delete
│   ├── adgroup-create
│   ├── adgroup-get
│   ├── adgroup-list
│   ├── adgroup-find
│   ├── adgroup-update
│   └── adgroup-delete
├── ads
│   ├── create
│   ├── get
│   ├── list
│   ├── find
│   ├── find-all
│   ├── update
│   └── delete
├── creatives
│   ├── create
│   ├── get
│   ├── list
│   └── find
├── apps
│   ├── search
│   ├── eligibility
│   ├── details
│   └── localized
├── product-pages
│   ├── list
│   ├── get
│   ├── locales
│   ├── countries
│   └── device-sizes
├── ad-rejections
│   ├── find
│   ├── get
│   └── find-assets
├── geo
│   ├── search
│   └── get
├── reports
│   ├── campaigns
│   ├── adgroups
│   ├── keywords
│   ├── searchterms
│   └── ads
└── impression-share
    ├── create
    ├── get
    └── list
```

## Resource commands

| Group | Purpose |
|---|---|
| `acls` | List accessible organizations and show the current API user. |
| `campaigns` | Create, query, update, and delete campaigns. |
| `budgetorders` | Create, query, and update budget orders. |
| `adgroups` | Create, query, update, and delete ad groups. `find-all` searches at organization level. |
| `keywords` | Manage targeting keywords. Bulk create, update, and delete commands accept JSON arrays. |
| `negatives` | Manage campaign and ad group negative keywords. |
| `ads` | Create, query, update, and delete ads. `find-all` searches at organization level. |
| `creatives` | Create and query organization-level creatives. |
| `apps` | Search apps, check eligibility, and get app details. |
| `product-pages` | Get custom product pages, locales, countries or regions, and preview device sizes. |
| `ad-rejections` | Get rejection reasons and app assets. |
| `geo` | Search locations or retrieve locations from a JSON request. |
| `reports` | Run campaign, ad group, keyword, search term, and ad reports. |
| `impression-share` | Create and check asynchronous impression share reports. |

## Create input

Create commands accept `--from-json` unless the table says it is required. Parent IDs used in request paths remain required with `--from-json`.

| Command | Required flags without `--from-json` |
|---|---|
| `campaigns create` | `--name`, `--adam-id`, `--daily-budget`, `--countries`, `--ad-channel-type`, `--billing-event`, `--supply-sources` |
| `budgetorders create` | `--from-json` |
| `adgroups create` | `--campaign-id`, `--name`, `--default-bid`, `--pricing-model`, `--start-time` |
| `keywords create` | `--campaign-id`, `--adgroup-id`, `--text` |
| `negatives campaign-create` | `--campaign-id`, `--text` |
| `negatives adgroup-create` | `--campaign-id`, `--adgroup-id`, `--text` |
| `ads create` | `--campaign-id`, `--adgroup-id`, `--creative-id`, `--name`, `--status` |
| `creatives create` | `--adam-id`, `--name`, `--type`; add `--product-page-id` for `CUSTOM_PRODUCT_PAGE` |

## Report options

The five report actions share these options. `adgroups`, `keywords`, `searchterms`, and `ads` require `--campaign-id`. The `keywords` and `searchterms` actions also accept `--adgroup-id`.

| Option | Description |
|---|---|
| `--start-time`, `--end-time` | Date range in `YYYY-MM-DD` form. Both are required without `--from-json`. |
| `--granularity` | `HOURLY`, `DAILY`, `WEEKLY`, `MONTHLY`, or `NONE`. The default is `DAILY`. |
| `--time-zone` | `ORTZ` or `UTC`. |
| `--group-by` | Comma-separated grouping dimensions. |
| `--selector-json`, `--selector` | Selector JSON inline, from `@file`, or from `@-`. |
| `--from-json` | Complete reporting request inline, from `@file`, or from `@-`. |
| `--return-row-totals` | Return row totals. Required with `--granularity NONE`. |
| `--return-grand-totals` | Return grand totals with `--granularity NONE`. |
| `--return-records-with-no-metrics` | Include records with no metrics. |
| `--limit`, `--offset` | Set selector pagination. Values must be between 0 and 1,000. |
| `--all` | Fetch every report page. |

For requests built from flags, totals cannot be combined with `HOURLY`, `DAILY`, `WEEKLY`, or `MONTHLY`. With `--from-json`, pagination flags update the request's `selector` when it is present.

## Impression share create

`impression-share create` accepts a complete request through `--from-json`, or it builds one from these options:

| Option | Description |
|---|---|
| `--name` | Report name, required and limited to 50 characters. |
| `--granularity` | `DAILY` or `WEEKLY`. The default is `DAILY`. |
| `--start-time`, `--end-time` | Paired dates for a daily report. Dates cannot precede `2020-04-12`, and the span cannot exceed 30 days. |
| `--date-range` | `LAST_WEEK`, `LAST_2_WEEKS`, or `LAST_4_WEEKS` for a weekly report. The default is `LAST_WEEK`. |
| `--selector-json`, `--selector` | Optional selector JSON inline, from `@file`, or from `@-`. |
| `--dry-run` | Print the request without sending it. |

Daily reports reject `--date-range`. Weekly reports reject start and end dates.

## Command help

Help is available through flags, the `help` command, or a group name by itself:

```bash
aads --help
aads help
aads campaigns
aads campaigns --help
aads campaigns help
aads help campaigns
aads campaigns create --help
aads help campaigns create
```
