module github.com/jsiebens/ionscale

go 1.21

require (
	github.com/99designs/keyring v1.2.2
	github.com/a-h/templ v0.2.543
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/bufbuild/connect-go v1.10.0
	github.com/caarlos0/env/v6 v6.10.1
	github.com/caddyserver/certmagic v0.20.0
	github.com/coreos/go-oidc/v3 v3.9.0
	github.com/dustinkirkland/golang-petname v0.0.0-20231002161417-6a283f1aaaf2
	github.com/glebarez/sqlite v1.10.0
	github.com/go-gormigrate/gormigrate/v2 v2.1.1
	github.com/go-jose/go-jose/v3 v3.0.1
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/hashicorp/go-bexpr v0.1.13
	github.com/hashicorp/go-multierror v1.1.1
	github.com/imdario/mergo v0.3.16
	github.com/jsiebens/go-edit v0.1.0
	github.com/jsiebens/mockoidc v0.1.0-rc2
	github.com/klauspost/compress v1.17.4
	github.com/labstack/echo-contrib v0.15.0
	github.com/labstack/echo/v4 v4.11.4
	github.com/libdns/azure v0.3.0
	github.com/libdns/cloudflare v0.1.0
	github.com/libdns/digitalocean v0.0.0-20230728223659-4f9064657aea
	github.com/libdns/googleclouddns v1.1.0
	github.com/libdns/libdns v0.2.1
	github.com/libdns/route53 v1.3.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/pointerstructure v1.2.1
	github.com/mr-tron/base58 v1.2.0
	github.com/nleeper/goment v1.4.4
	github.com/ory/dockertest/v3 v3.10.0
	github.com/prometheus/client_golang v1.18.0
	github.com/rodaine/table v1.1.0
	github.com/sony/sonyflake v1.2.0
	github.com/spf13/cobra v1.8.0
	github.com/stretchr/testify v1.8.4
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a
	github.com/travisjeffery/certmagic-sqlstorage v1.1.1
	github.com/xhit/go-str2duration/v2 v2.1.0
	go.uber.org/zap v1.26.0
	golang.org/x/crypto v0.18.0
	golang.org/x/net v0.20.0
	golang.org/x/oauth2 v0.16.0
	golang.org/x/sync v0.6.0
	google.golang.org/protobuf v1.32.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
	gorm.io/plugin/prometheus v0.1.0
	inet.af/netaddr v0.0.0-20230525184311-b8eac61e914a
	tailscale.com v1.56.1
)

require (
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.9.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.5.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/alexbrainman/sspi v0.0.0-20231016080023-1a75b4708caa // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.26.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.36.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/continuity v0.4.3 // indirect
	github.com/coreos/go-iptables v0.7.0 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dblohm7/wingoes v0.0.0-20240108175832-49174e152ce1 // indirect
	github.com/digitalocean/godo v1.107.0 // indirect
	github.com/docker/cli v25.0.2+incompatible // indirect
	github.com/docker/docker v25.0.2+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.6.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/glebarez/go-sqlite v1.22.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang-jwt/jwt/v5 v5.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/nftables v0.1.1-0.20230115205135-9aa6fdf5a28c // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.5.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.5 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.1 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/native v1.1.1-0.20230202152459-5c7d0dd6ab86 // indirect
	github.com/jsimonetti/rtnetlink v1.4.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/lib/pq v1.10.6 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/socket v0.5.0 // indirect
	github.com/mholt/acmez v1.2.0 // indirect
	github.com/miekg/dns v1.1.57 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc6 // indirect
	github.com/opencontainers/runc v1.1.12 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tailscale/netlink v1.1.1-0.20211101221916-cabfb018fe85 // indirect
	github.com/tkuchiki/go-timezone v0.2.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/zeebo/blake3 v0.2.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.46.1 // indirect
	go.opentelemetry.io/otel v1.21.0 // indirect
	go.opentelemetry.io/otel/metric v1.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.21.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go4.org/intern v0.0.0-20230525184215-6c62f75575cb // indirect
	go4.org/mem v0.0.0-20220726221520-4f986261bf13 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.17.0 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	google.golang.org/api v0.155.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240108191215-35c7eff3a6b1 // indirect
	google.golang.org/grpc v1.60.1 // indirect
	gotest.tools/v3 v3.4.0 // indirect
	modernc.org/libc v1.40.1 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.7.2 // indirect
	modernc.org/sqlite v1.28.0 // indirect
	nhooyr.io/websocket v1.8.10 // indirect
)
