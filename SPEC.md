nccald
======

nccald is a simple daemon to provide calendar notifications for Namecoin name
expirations.

The daemon periodically queries a Namecoin Core instance using the `name_list`
RPC command and can do one or both of the following:

  - Generate an ICS calendar file listing names with their projected expiration
    dates
  - Push to a CalDav server
  - Future: Generate emails, execute commands?

Proposed interface
------------------

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
does change).

Calculation of expiration dates
-------------------------------

Assume 10 minute block time and calculate expiration date accordingly, rounded
to a multiple of a configurable time interval (the quantum), and then has a
configurable amount of time (the margin) subtracted from it.

The important criterion here is to avoid “date jump”; where the projected expiration
date keeps jumping from one calendar day to another. However, as the 10 minute
block time is only an average, as expiration approaches, the estimated date of
expiry may change. Thus the idea is to round the estimated date of expiry *down*
to e.g. a multiple of N days, then subtract M days as an additional margin
(according to how close to expiry someone wants to renew their names.)
