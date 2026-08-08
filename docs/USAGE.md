# Usage

## Configure authentication

Run the setup command and enter the Apple Ads API credential identifiers, organization ID, private key path, and optional default currency:

```bash
aads configure
```

The command writes `~/.aads/config.yaml`. The directory is private to your user and the file is readable only by your user. The config stores the private key path, not the key contents or access tokens.

You can also set the values through the environment:

| Variable | Value |
|---|---|
| `AADS_CLIENT_ID` | OAuth client ID |
| `AADS_TEAM_ID` | Team ID |
| `AADS_KEY_ID` | Private key ID |
| `AADS_ORG_ID` | Apple Ads organization ID |
| `AADS_PRIVATE_KEY_PATH` | Path to an EC P-256 private key |
| `AADS_DEFAULT_CURRENCY` | Default ISO 4217 currency code |
| `AADS_CURRENCY` | Alias for `AADS_DEFAULT_CURRENCY` |

Command flags take precedence over environment variables. Environment variables take precedence over the config file.

The private key may use PKCS#8 or SEC1 PEM encoding. `aads` creates an ES256 client secret, exchanges it for an OAuth access token, and keeps that token in memory until it needs to refresh it.

## Select an organization

The configured organization applies to every API request. Override it for one command with `--org-id`:

```bash
aads --org-id 1234567 campaigns list
```

Use `aads acls list` to see the organizations available to the current API user.

## Provide request data

Create and update commands accept JSON inline, from a file, or from standard input. Reports accept complete request objects the same way:

```bash
aads keywords create --campaign-id 12345 --adgroup-id 67890 \
  --from-json '[{"text":"grammar","matchType":"BROAD"}]'
aads keywords create --campaign-id 12345 --adgroup-id 67890 --from-json @keywords.json
cat keywords.json | aads keywords create --campaign-id 12345 --adgroup-id 67890 --from-json @-
```

Find commands accept a complete selector with `--selector-json`. Simple selectors can use `--field`, `--op`, and `--values`:

```bash
aads campaigns find --field status --op EQUALS --values ENABLED
aads campaigns find --selector-json @selector.json
```

Use `--fields` on endpoints that support partial fetch:

```bash
aads campaigns list --fields id,name,status
```

## Create resources with flags

Most create commands accept either typed flags or `--from-json`. When you use `--from-json`, IDs that select a parent resource are still required.

| Command | Required typed input |
|---|---|
| `campaigns create` | `--name`, `--adam-id`, `--daily-budget`, `--countries`, `--ad-channel-type`, `--billing-event`, and `--supply-sources` |
| `adgroups create` | `--campaign-id`, `--name`, `--default-bid`, `--pricing-model`, and `--start-time` |
| `keywords create` | `--campaign-id`, `--adgroup-id`, and `--text` |
| `negatives campaign-create` | `--campaign-id` and `--text` |
| `negatives adgroup-create` | `--campaign-id`, `--adgroup-id`, and `--text` |
| `ads create` | `--campaign-id`, `--adgroup-id`, `--creative-id`, `--name`, and `--status` |
| `creatives create` | `--adam-id`, `--name`, and `--type`; `CUSTOM_PRODUCT_PAGE` also needs `--product-page-id` |
| `budgetorders create` | `--from-json` |

Campaign channel types are `SEARCH` and `DISPLAY`. Billing events are `TAPS` and `IMPRESSIONS`. Supply sources are `APPSTORE_SEARCH_RESULTS`, `APPSTORE_SEARCH_TAB`, `APPSTORE_TODAY_TAB`, and `APPSTORE_PRODUCT_PAGES_BROWSE`.

Ad group pricing models are `CPC` and `CPM`; start time accepts an ISO date or date-time. Keyword and negative keyword match types are `BROAD` and `EXACT`. Ad status is `ENABLED` or `PAUSED`. Creative type is `DEFAULT_PRODUCT_PAGE` or `CUSTOM_PRODUCT_PAGE`.

## Handle money

When `aads` builds a money value from flags, it resolves the currency in this order:

1. `--currency`
2. `AADS_DEFAULT_CURRENCY`, or its `AADS_CURRENCY` alias
3. `default_currency` in the config file
4. the matching organization returned by `GET /acls`

If the currency cannot be resolved to one value, the command stops before sending the request. JSON request bodies may include Apple money objects directly.

## Paginate results

List and find commands use `--limit` and `--offset` for one page. Pass `--all` to keep requesting pages in server order:

```bash
aads campaigns list --limit 50 --offset 100
aads adgroups list --campaign-id 12345 --all
```

Automatic pagination keeps the initial offset and stops at the reported total, after a short page, or when Apple returns no progress. It requests at most 1,000 pages.

## Choose output

JSON is the default. Table and YAML output are also available:

```bash
aads campaigns list --output json
aads campaigns list --output table
aads campaigns list --output yaml
```

Output contains the Apple response `data` value by default. Pass `--envelope` when you also need pagination, metadata, or report totals. Normal output goes to standard output. Progress, retry notices, and errors go to standard error.

## Preview and confirm changes

Mutation commands accept `--dry-run`. It prints the resolved HTTP method, path, redacted headers, and request body without calling Apple:

```bash
aads campaigns update --id 12345 --status PAUSED --dry-run
```

Delete and bulk-delete commands ask for confirmation in an interactive terminal. Pass `--yes` to confirm in scripts. Noninteractive deletion fails unless `--yes` is present.

## Retries and timeouts

API requests time out after 30 seconds. When Apple returns `401`, `aads` refreshes the access token and retries once.

The client may retry `429` and `5xx` responses after waiting 2, 4, 8, then 16 seconds. It honors `Retry-After` when Apple sends it. Retries apply to GET, PUT, DELETE, selector find requests, reports, and bulk deletes. Create requests are not retried because repeating one may create a duplicate resource.

Press Control-C to cancel a request, pagination run, or retry wait.

## Reports

Standard reports cover campaign, ad group, keyword, search term, and ad scopes. Every scope except `campaigns` requires `--campaign-id`. Use `--adgroup-id` with keyword or search term reports to select their ad group endpoint.

```bash
aads reports campaigns \
  --start-time 2026-07-01 \
  --end-time 2026-07-31 \
  --time-zone ORTZ \
  --all

aads reports keywords \
  --campaign-id 12345 \
  --adgroup-id 67890 \
  --from-json @report.json
```

Without `--from-json`, `--start-time` and `--end-time` are required. Dates use `YYYY-MM-DD`. Granularity may be `HOURLY`, `DAILY`, `WEEKLY`, `MONTHLY`, or `NONE`; the default is `DAILY`. Time zone may be `ORTZ` or `UTC`.

Use `--selector-json` or its `--selector` alias for a selector. `--group-by` accepts comma-separated dimensions. `--return-records-with-no-metrics` includes empty rows. `--limit` and `--offset` must be between 0 and 1,000. Pass `--all` to fetch all report pages.

When the CLI builds the request from flags, totals require `--granularity NONE`. `--return-row-totals` is required in that mode, and `--return-grand-totals` is optional. Row totals and grand totals cannot be used with another granularity.

A complete request passed through `--from-json` bypasses the typed request-body fields. Scope IDs remain in the command flags. Pagination flags still update the request's `selector` object when one is present.

## Impression share reports

Impression share reports use the asynchronous custom-report endpoints. Create a report, then use `get` or `list` to check it.

```bash
aads impression-share create \
  --name "July share" \
  --granularity DAILY \
  --start-time 2026-07-01 \
  --end-time 2026-07-31 \
  --selector @selector.json

aads impression-share create \
  --name "Recent share" \
  --granularity WEEKLY \
  --date-range LAST_2_WEEKS
```

The name is required and may contain up to 50 characters. A `DAILY` report accepts paired start and end dates from 2020-04-12 onward, with a maximum span of 30 days. A `WEEKLY` report uses `--date-range` instead of dates. Its choices are `LAST_WEEK`, `LAST_2_WEEKS`, and `LAST_4_WEEKS`; the default is `LAST_WEEK`.

Both granularities accept `--selector-json` or `--selector`. Pass a complete request with `--from-json` when you do not want the CLI to build it from flags.

## Help aliases

These forms show the same help:

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
