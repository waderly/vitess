# Reparenting

This document describes the reparenting features of Vitess. Reparenting is used when the master for a Shard is
changing from one host to another. It can be triggered (for maintenance for instance) or happen automatically
(based on the current master dying for instance).

Two main types of reparenting supported by Vitess are Active Reparents (the Vitess toolchain is handing it all)
and External Reparents (another tool is responsible for reparenting, and the Vitess toolchain just update its
internal state.

## Active Reparents

They are triggered by using the 'vtctl ReparentShard' command. See the help for that command. It currently doesn't use transaction GroupId.

## External Reparents

In this part, we assume another tool has been reparenting our servers. We then trigger the
'vtctl ShardExternallyReparented' command.

The flow for that command is as follows:
- the shard is locked in the global topology server.
- we read the Shard object from the global topology server.
- we read all the tablets in the replication graph for the shard. We also check the new master is in the map. Note we allow partial reads here, so if a data center is down, as long as the data center containing the new master is up, we keep going.
- we call the 'SlaveWasPromoted' remote action on the new master. This remote action makes sure the new master is not a MySQL slave of another server (the 'show slave status' command should not return anything, meaning 'reset slave' should ave been called).
- for every host in the replication graph, we call the 'SlaveWasRestarted' action. It takes as paremeter the address of the new master. On each slave, it executes a 'show slave status'. If the master matches the new master, we update the topology server record for that tablet with the new master, and the replication graph for that tablet as well. If it doesn't match, we keep the old record in the replication graph (pointing at whatever master was there before). We optionally Scrap tablets that bad (disabled by default).
- if a smaller percentage than a configurable value of the slaves works (80% be default), we stop here.
- we then update the Shard object with the new master.
- we rebuild the serving graph for that shard. This will update the 'master' record for sure, and also keep all the tablets that have successfully reparented.

Optional Flags:
- -accept-success-percents=80: will declare success if more than that many slaves can be reparented
- -continue_on_unexpected_master=false: if a slave has the wrong master, we'll just log the error and keep going
- -scrap-stragglers=false: will scrap bad hosts

Failure cases:
- The global topology server has to be available for locking and modification during this operation. If not, the operation will just fail.
- If a single topology server is down in one data center 9and it's nto the master data center), the tablets in that data center will be ignored by the reparent. Provided it doesn't trigger the 80% threshold, this is not a big deal. When the topology server comes back up, just re-run 'vtctl InitTablet' on the tablets, and that will fix their master record.
- If scrap-straggler is false (the default), a tablet that has the wrong master will be kept in the replication graph with its original master. When we rebuild the serving graph, that tablet won't be added, as it doesn't have the right master.
- if more than 20% of the tablets fails, we don't update the Shard object, and don't rebuild. We assume something is seriously wrong, and it might be our process, not the servers. Figuring out the cause and re-running 'vtctl ShardExternallyReparented' should work.
- if for some reasons none of the slaves report the right master (replication is going through a proxy for instance, and the master address is not what the clients are showing in 'show slave status'), the result is pretty bad. All slaves are kept in the replication graph, but with their old (incorrect) master. Next time a Shard rebuild happens, all the servers will disappear. At that point, fixing the issue and then re-parenting will work.

## Reparenting And Serving Graph

When reparenting, we shuffle servers around. A server may get demoted, another promoted, and some servers may end up with the wrong master in the replication graph, or scrapped.

It is important to understand that when we build the serving graph, we go through all the servers in the replication graph, and check their masters. If their master is the one we expect (because it is in the Shard record), we keep going and add them to the serving graph. If not, they are skipped, and a warning is displayed.

When such a slave with the wrong master is present, re-running 'vtclt InitTablet' with the right parameters will fix the server. So the order of operations should be to fix mysql replication, make sure it is caught up, run 'vtctl InitTablet', and maybe restart vttablet if needed.

Alternatively, if another reparent happens, and the bad slave recovers and now replicates from the new master, it will be re-added, and resume proper operation.

The old master for reparenting is a specific case. If it doesn't have the right master during the reparent, it will be scrapped (because it's not in the replication graph at all, so it would get lost anyway).
