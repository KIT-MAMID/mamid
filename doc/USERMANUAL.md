# Usermanual for MAMID

## Introduction

The web interfaces of MAMID gives information about the status of MAMID:
 - status of slaves and Replica Sets
 - occurring problems

It enables the user to configure MAMID:
 - create and modify slaves
 - create and modify Replica Sets
 - assign slaves into risk groups

A button to access the help function can be found at the top right corner.
		
## Overview

 - General information: The overview shows the three most recent problems, as well as the  general status of the slaves.

 - Problems:
        The three most recent problems will appear in the overview. Problems can occur in slaves or Replica Sets. 
         
 - Slave State:
        A pie diagram of the slaves states. Slaves can have one of four states: 
	- Active (green)
	- Maintenance (blue)
	- Disabled (grey)
	- Problematic (red)


## Problems

- General information
    - Problems can occur in slaves and Replica Sets. 
    - Every problem shown will have a link to the affected slave or Replica Set.
    - Problems are ordered chronological.

## Slaves

Slaves are ordered by their state.

 - General information:
    - Slaves have four parameters and four states.
    - While setting up a slave, the number of monogods can be choosen. 
    - In the top of each slaves view, the number of possible and already deployed slaves is listed.
    - Parameters:
       	 1. Hostname: a name to identify the slave
         2. Slave port: The port on the host that should be used by the slave.
         3. Mongod port range: the ports to be used by spawned instances of mongod
         4. Persistent storage: Choose this option if the host has and the slave is meant to use persistent storage
    - States:
        1. Active: 
        	Slaves can be set to active.
                The slave is available to host mongod instances as part of Replica Sets. Active slaves will be monitored. If a problem occurs in an active slave, it will be set to problematic.
        2. Maintenance: 
                Slaves can be set to maintenance mode.
                A slave in maintenance mode will not be monitored and no new mongod instances will be spawned. Existing mongods will be left untouched.
		Slaves in maintenance mode can be reconfigured.
        3. Disabled: 
                Slaves can be set to disabled state.
		Marks the slave as not running any mongod instances. 
        4. Problematic: 
                If a problem occurs in a slave, it will be set to problematic. This happens automatically.
 - Possible Problems:
    1. Slave is unreachable: Occurs when a slave does not respond. 
	   Check if a slave instance of MAMID is running at the specified port.
	   Check if the associated host is running as planned.
 - Create slave:
    1. Use [Create new slave] in the top right corner of the slave view or [New Slave] in the sidebar. 
    2. Set the parameters of the slave.
    3. Choose [Apply].
    
    *After creation, a slave is in disabled state.*
 
 - Change slave state:
    1. Choose the slave to change in the Slaves view. Slaves are sorted by their state and can be filtered by their hostname.
    2. In [Slave Control], one of three states can be set.
        a. Active
        b. Maintenance
        c. Disabled

 - Modify slave settings:
    1. Choose the slave to change in the Slaves view. Slaves are sorted by their state and can be filtered by their hostname.
    2. Change the state of the slave to [Maintenance] (s. Change slave state)
    3. In Slave Settings, the parameters of the slave can be modified.
    4. Choose [Apply] 
 
 - Remove slave:
     1. Choose the slave to remove in the Slaves view. Slaves are sorted by their state and can be filtered by their hostname.
     2. In Slave Control, choose [Disabled] 
     3. In the pop-up, click [Disable slave]
     4. In the bottom of the slaves view, choose [Remove slave from system]
     5. In the pop up, click [Remove slave]
	
## Risk Groups

 - General information:
    Risk groups are sorted by their time of creation.
    They reduce downtimes of Replica Sets.
    To show the slaves in a risk group, click on the risk groups title.

 - How to choose a risk group: When ever slaves share a common fault source, they should be assigned to different risk groups. If this is not possible, the fault sources most likely to fail should e considered first. Possible fault sources are e.g. slaves running on the same blade or having the same power source. 
   
 - Create Risk Group:
    1. In Create a new Risk Group, type in a name for the risk group.
    2. Choose [Create]
    
   *It takes a few seconds until the new risk group is shown*

 - Assign slaves to a Risk group:
    1. Disable the slave.
    2. Choose the slave from the list of unassigned slaves or another risk group or search by their hostname.
    3. In the drop-down menu on the right side of the slaves field, choose the desired risk group.

 - Remove slave from a Risk Group:
    1. Disable the slave. 
    2. Open the Risk Group.
    3. Find the slave to be removed. Slaves can be filtered by their hostname.
    4. In the drop-down menu on the right side of the slaves field, choose [Remove from Risk Group]
 
 - Remove a Risk Group:
    1. Remove all slaves from the risk group
    2. Click on the red bin icon at the right side of the risk groups title field.
    3. In the pop-up window, choose [Remove Risk Group]


## Replica Sets

 - General information:
	- A Replica Set consist of a number of volatile and persistent slaves. Every Replica Set stores one set of data.
	- Each Replica Set has four parameters.
	- A Replica Set has one of three [Sharding](https://docs.mongodb.com/manual/sharding/ "Sharding in MongoDB") settings:
		1. [configsvrMode](https://docs.mongodb.com/manual/reference/program/mongod/#cmdoption--configsvr)
		2. [shardsvr](https://docs.mongodb.com/manual/reference/program/mongod/#cmdoption--shardsvr)
		3. none: no sharding will be apllied
    	- Replica sets are listed divided by working sets and sets with problems.
        - To get more information about each Replica Set, click on its entry in the list.
 
- Replica set view:
	- Problems in the replica set are shown at the top.
	- Replica set overview: List of Mongods assigned to the replica set. Its slave is linked and problems are shown.

 - Parameters:
        1. Replica Set name: A name to identify the Replica Set.
        2. Persistent nodes: The number of persistent nodes in the Replica Set.
        Needs to be zero or a positive integer.
        3. Volatile nodes: The number of volatile nodes in the Replica Set.
        Needs to be zero or a positive integer.
        The sum of persistent and volatile nodes needs to be odd.
        4. Sharding configuration server: 
       
 - Possible problems:
    - Degraded Replica Set: Occurs when one or more instances of mongod from the slaves in the Replica Set are not running.
    Check the status of the assigned slaves.
    - Unsatisfied constraints: Occurs when there are not enough ports to spawn the specified number of mongods for the Replica Set.
    Check if there are enough free ports in assigned slaves.
    Check if there are disabled slaves assigned to the Replica Set.
   
 - Create Replica Set:
    1. Choose [Create new Replica Set] in the top right corner of the Replica Sets view or the shortcut [New Replica Set] in the sidebar
    2. Fill in the risk groups parameters.
    3. Choose the sharding setting.
    4. Click [Create]
       
 - Modify Replica Set:
    Only the number of persistent and volatile nodes can be changed.

    1. Choose the Replica Set from the list in the Replica Sets view.
    2. Find [Replica Set Settings]
    3. Change the number of volatile and persistent nodes.
    They need to be zero or a positive integer.
    The sum of persistent and volatile nodes needs to be odd.
 
 - Remove Replica Set:
    Attention: Removing a Replica Set will drop all data stored in it.
    Recovering the data is not possible.
	   
    After removing a Replica Set, all slaves assigned to it are free to be used by other Replica Sets.

    1. Choose the Replica Set from the list in the Replica Sets view.
    2. Click [Remove Replica Set from system]. 
    3. In the pop-up window, choose [Remove Replica Set**]
    
## System

- General information:
	- MongoDB Key File: All Mongod instances in the cluster are deployed with a key file for Internal Authentication
	- Administrative Account: All Mongod instances deployed by MAMID have user access control enabled. Replica Sets have an administrative user with the root role. You may use this user for further configuration, e.g. adding additional users or configuring mongos instances.
