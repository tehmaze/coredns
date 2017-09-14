# geo

*geo* enables GeoIP enabled DNS records. Configuration is done by supplying
a YaML file with the domain and service definitions.

The implementation is a port of the PowerDNS geoip backend and the
configuration syntax is (mostly) compatible.

## Syntax

```
geo <filename>
```

* **filename** is the absolute path to the YaML configuration file.

## Examples

This is an example configuration offering continent-based Geo DNS resolution
on the `geo.example.org` domain:

```
database: /etc/coredns/GeoLite2-Country.mmdb
domains:
- domain: example.org
  ttl:    60
  records:
    # Zone records
    example.org:
      - soa: sns.dns.icann.org. noc.dns.icann.org. 2017042783 7200 3600 1209600 3600
      - ns: a.iana-servers.net.
      - ns: b.iana-servers.net.
    
    # This is our fallback, we include all nodes
    unknown.example.org:
      - a: 192.0.2.1
      - a: 198.51.100.1
      - a: 203.0.113.1

    # Africa
    af.example.org:
      - a: 192.0.2.1

    # Antartica
    an.example.org:
      - a: 198.51.100.1

    # Asia
    as.example.org:
      - a: 192.0.2.1

    # Europe
    eu.example.org:
      - a: 203.0.113.1

    # North America
    na.example.org:
      - a: 198.51.100.1 

    # Oceania
    oc.example.org:
      - a: 198.51.100.1

    # South America
    sa.example.org:
      - a: 198.51.100.1 

  services:
    geo.example.org: '%cn.example.org'
```

## Service placeholders

The following placeholders are available:

    %af     IP address family, "v4" for IPv4 and "v6" for IPv6
    %co     ISO-3166-1 alpha-2 country code
    %cn     Continent code
    %hh     2-digit hour
    %yy     4-digit year
    %wds    Weekday string
    %wd     Weekday number (0 = Sunday)
    %mos    Month string
    %mo     Month number (1 = January)
    %ip     IP address
    %%      Literal %

Currently not supported:

    %as     AS number
    %re     Region name
    %na     Name
    %ci     City name

If any of the placeholders can't be resolved, it defaults to "unknown". 
