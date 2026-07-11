# htee

A fast, first-class terminal HTTP client, written in Go. `htee` transliterates
[httpie](https://httpie.io/)'s CLI grammar into a single static binary
called `ht` — same item syntax, same flags, same output model — plus one
addition: an `HT_AUTH` (or `AUTH_TOKEN`) environment variable for
auto-injecting a bearer token without typing `-a` on every call.

## Install

```sh
make build     # builds bin/ht
make install   # installs ht to $GOBIN (or $GOPATH/bin)
```

## Usage

```
ht [METHOD] URL [ITEM ...]
```

`METHOD` is optional and inferred: no items (or only header/query items)
infers `GET`; any body-data item infers `POST`.

```sh
ht example.org                       # GET
ht POST example.org foo=bar n:=1     # POST, JSON body {"foo":"bar","n":1}
ht PUT example.org/1 name=Bob
ht DELETE example.org/1
ht example.org X-Custom:value id==42 # header + query string item
```

### Item syntax

| Separator | Meaning                                    | Example              |
|-----------|---------------------------------------------|-----------------------|
| `=`       | JSON/form string field                      | `foo=bar`             |
| `:=`      | JSON field with a raw (non-string) value    | `n:=1`, `ok:=true`    |
| `==`      | Query string parameter                      | `page==2`             |
| `:`       | Header                                      | `X-Custom:value`      |
| `:` empty | Remove a default header                     | `Accept:`             |
| `@`       | File upload (multipart)                     | `avatar@photo.png`    |
| `=@`      | Field value loaded from file content        | `bio=@bio.txt`        |
| `:=@`     | JSON field value loaded from file content   | `data:=@data.json`    |

Nested JSON via bracket paths is supported: `person[name]=bob`,
`tags[]=a`, `tags[]=b`.

### Auth

```sh
ht -a user:pass example.org           # Basic
ht -a token -A bearer example.org     # Bearer
ht -a user -A digest example.org      # Digest
ht https://user:pass@example.org      # URL-embedded userinfo
HT_AUTH=$TOKEN ht example.org         # auto Authorization: Bearer $TOKEN
AUTH_TOKEN=$TOKEN ht example.org      # same, used if HT_AUTH isn't set
```

Precedence: explicit `-a`/`-A` > URL-embedded userinfo > `HT_AUTH` >
`AUTH_TOKEN` > `.netrc` (skip with `--ignore-netrc`) > none.

### Redirects, SSL, and network options

```sh
ht -F example.org/redirects-somewhere      # follow 30x Location redirects
ht -F --max-redirects 5 example.org        # cap the number of hops followed
ht --all -F -v example.org                 # show every hop's request/response
ht --verify no https://self-signed.local   # skip SSL certificate verification
ht --verify /path/to/ca-bundle.pem https://internal.example.org
ht --ssl tls1.2 https://example.org        # pin the TLS protocol version
ht --cert client.pem --cert-key client.key https://example.org
ht --timeout 5 example.org                 # error if the request takes over 5s
ht --proxy http:http://localhost:8080 example.org
```

`--verify` accepts `yes`/`no` or a CA bundle file path. `--ssl` accepts
`ssl2.3` (default: negotiate the highest mutually supported protocol),
`tls1`, `tls1.1`, `tls1.2`, or `tls1.3` (`ssl3` is rejected - unsupported by
Go's TLS stack). `--ciphers` takes Go `crypto/tls` cipher suite names
(colon- or comma-separated), not OpenSSL names. `--max-headers` is accepted
for httpie compatibility but isn't enforced.

### Output

By default `ht` prints the full request and response (headers and body).
Use `-p/--print` with a string of `H`,`B`,`h`,`b`,`m` to select exactly
what's shown (request headers/body, response headers/body/meta), or the
shortcuts `-h`/`-b`/`-m`. `--offline` builds and prints the request without
sending it.

Response bodies are pretty-printed and colorized by mimetype
(`--pretty`, `--style`, `--format-options`); `-S/--stream` disables
re-formatting and writes the body as it arrives.

## Development

```sh
make test   # go test ./...
make vet    # go vet ./...
make fmt    # gofmt -l . (fails if anything is unformatted)
make clean  # remove bin/ and build cache
```
