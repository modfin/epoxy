# epoxy
Reverse proxy and/or static file server, with the primary goal of serving a self-hosted web 
application behind a Cloudflare Zero Trust tunnel. Validates Cloudflare `Cf-Access-Jwt-Assertion` JWT token. Can also be used as a generic reverse proxy or static file server.

## Usage
Settings are defined through environment variables.
### What to serve
#### Reverse proxy (optional, enables reverse proxy if defined)
* `ROUTES` format
  ```
  Prefix      /backend-0   http://backend-0:8080
  PrefixStrip /backend-1   http://backend-1:8080
  ```
  Where `PrefixStrip` strips the matching prefix before reverse proxying the request to the backend.

#### Static file server (optional)
* `PUBLIC_DIR` directory to serve static files from, e.g. `./public`
* `PUBLIC_PREFIX` path where the web application expects to find static files.

### Server modes
All different types of server modes can be combined at the same time, on different ports.
#### Server without authentication (optional)
In this mode epoxy can be used as a regular reverse proxy or static file server.
* `NO_AUTH_ENABLE` enables no auth server.
* `NO_AUTH_ADDR` address to serve at, e.g. `":8080"` or `"127.0.0.1:8080"`

#### Server for requests coming from [cloudflared](https://github.com/cloudflare/cloudflared) tunnel
* `CF_ADDR` address to serve at, e.g. `":8080"` or `"127.0.0.1:8080"`
* `CF_JWKS_URL` Cloudflare JWKS Url to [validate JWT](https://developers.cloudflare.com/cloudflare-one/identity/authorization-cookie/validating-json/)\
e.g. `https://<your-team-name>.cloudflareaccess.com/cdn-cgi/access/certs`
* `CF_APP_AUD` Cloudflare Application Audience (AUD) Tag.

#### Dev mode server
* `DEV_ADDR` address to serve at, e.g. `":8080"` or `"127.0.0.1:8080"`
* `DEV_ALLOWED_USER_SUFFIX` allowed user suffix e.g. `@test.com`, will be used in generated JWT as subject.
* `DEV_BCRYPT_HASH` dev authentication password bcrypt hash, to generate:\
`htpasswd -bnBC 10 "" "[PASSWORD]" | tr -d ':\n'`
* `DEV_SESSION_DURATION` in standard go time.Duration format e.g. 10m, 1h, 24h

### Misc
#### Fetch external JWT
After validating `Cf-Access-Jwt-Assertion` header, contact external/custom service passing along the `Cf-Access-Jwt-Assertion` header. Can be used for fetching extended info about the user that is logged into zero trust.
* `EXT_JWKS_URL` JWKS url with public keys for validating the new token received from the external service.
* `EXT_JWT_URL` URL to fetch from.
* `EXT_JWT_SUBJECT_PATH` path in external claims to grab subject for epoxy token below.

#### JWT Keys
After fetching external JWT or always in *dev mode*, a new JWT token is generated and sent in the `Epoxy-Token` header.
* `JWT_EC_256` used for generating JWT and *dev mode* cookie
* `JWT_EC_256_PUB` used for verifying *dev mode* cookie. 
