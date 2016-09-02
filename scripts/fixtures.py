#!/usr/bin/env python2
import requests
import argparse


def createSlave(api, ip):
    # Create slave
    j = requests.put(api + "/slaves", json={"configured_state":"disabled","hostname":ip,"slave_port":8081,"mongod_port_range_begin" :18080,"mongod_port_range_end":18081}).json()
    # Set slave to active
    print("Creating slave {}".format(ip), requests.post(api + "/slaves/" + str(j["id"]), json={"id":j["id"],"hostname":ip,"slave_port":8081,"mongod_port_range_begin":18080,"mongod_port_range_end":18081,"persistent_storage":False,"configured_state":"active","configured_state_transitioning":False,"risk_group_id":None}))

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-m", "--master", type=str, default="10.101.202.1", help="IP/hostname of the master")
    parser.add_argument("-s", "--https", action="store_const", const="https", default="http", help="Use https")
    actions = parser.add_mutually_exclusive_group(required=True)
    actions.add_argument("-c", "--createSlaves", action="store_true", help="creates slaves for docker slaves and activates them")
    args = parser.parse_args()

    api = "{}://{}:8080/api".format(args.https, args.master)

    if args.createSlaves:
        for i in range(1, 4):
            createSlave(api, "10.101.202.1{:02d}".format(i))
