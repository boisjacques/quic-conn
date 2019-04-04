#!/usr/bin/env python
# -*- coding: utf-8 -*-

import sys
import os


def main():
    if len(sys.argv) != 2:
        return -1

    interfaces = ["enp0s8", "enp0s9", "enp0s10"]
    str_number = sys.argv[1]
    number = int(str_number)
    inverted = 3 - number

    os.system("sudo tc qdisc del dev enp0s3 root")
    os.system("sudo tc qdisc add dev enp0s3 root netem delay 50ms")

    for interface in interfaces:
        os.system("sudo ip link set " + interface + " down")

    if number > 1:
        for i in range(0, number - 1):
            os.system("sudo ip link set " + interfaces[i] + " up")
            os.system("sudo tc qdisc del dev " + interfaces[i] + " root")
            os.system("sudo tc qdisc add dev " + interfaces[i] + " root netem delay 50ms")

    return 0


if __name__ == "__main__":
    main()
