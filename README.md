nccald
======

nccald is a simple daemon to provide calendar notifications for Namecoin name
expirations.

The daemon periodically queries a Namecoin Core instance using the `name_list`
RPC command and can do one or both of the following:

  - Generate an ICS calendar file listing names with their projected expiration
    dates
  - Push to a CalDav server


Usage
-----

Daemon command line arguments:

    nccald
      -service.xyz=etc.
      -nccald.calmargin=72h             Interval of uncertainty to allow in projected expiration calculation.
      -nccald.calquantum=72h            Projected expiration time is rounded to a multiple of this time interval.
      -nccald.calqueryinterval=10m      Query interval.
      -nccald.icspath=...               Write ICS calendar file to this path if specified.
      -nccald.caldavurl=...             Specify URL of a CalDAV calendar resource to update a CalDAV calendar if desired.
      -nccald.caldavusername=...        Username to use for CalDAV authentication.
      -nccald.caldavpassword=...        Password to use for CalDAV authentication.
      -nccald.namecoinrpcaddress=...    Address of Namecoin RPC server (e.g. "127.0.0.1:8336").
      -nccald.namecoinrpcusername=...   Username to use for Namecoin RPC.
      -nccald.namecoinrpcpassword=...   Password to use for Namecoin RPC.
      -nccald.namecoinrpccookiepath=... Path to cookie file to use for Namecoin RPC.
      -nccald.once=1                    Write ICS file/update CalDAV once and exit.

When the ICS calendar file is written, the file `${PATH}.tmp` is first written
and renamed over the original file (thus writes are atomic but inode number
does change); therefore you can use inotify to watch for changes to the ICS
file if you like.

Here is example usage for generation of an ICS file periodically:

    nccald -nccald.namecoinrpccookiepath=~/.namecoin/.cookie -nccald.icspath=~/namecoin.ics

Here is example usage for updating a CalDAV resource periodically. Note that the URL given should be to a CalDAV calendar resource:

    nccald
      -nccald.namecoinrpccookiepath=~/.namecoin/.cookie
      -nccald.caldavurl=https://example.com/calendars/my-calendar/
      -nccald.caldavusername=calendaruser
      -nccald.caldavpassword=calendarpassword

You can use `-nccald.once=1` if you want to run a cron job instead of a persistent daemon:

    nccald -nccald.once=1 ...

For debug logging, pass `-xlog.severity=debug`.

For a full list of options, run `nccald -help`.

Calculation of expiration dates
-------------------------------

Since Namecoin name expiration is based on block height, exact calculations of
a chronological expiration time are not possible. nccald calculates a
conservative (that is, early) expiration time based on configurable margin and
quantum values.

The expiration date of a name is guessed based on assuming each block averages
10 minutes; then the margin duration is subtracted. Finally, the resulting time
is rounded *down* to a multiple of the quantum duration; this is to ensure that
the estimated expiry date does not constantly fluctuate between different
values as the block height changes.

If you want to increase your safety margin of how far in advance you renew
names, increase `-nccald.calmargin`. If you are noticing “date jump” in which a
projected expiration keeps moving from one calendar day to another, increase
`-nccald.calquantum`.

Licence
-------

    © 2021 Hugo Landau <hlandau@devever.net>        GPLv3+ License

