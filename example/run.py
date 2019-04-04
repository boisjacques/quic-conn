#!/usr/bin/env python
# -*- coding: utf-8 -*-

import sys
import os


def main():
    if len(sys.argv) != 2:
        print "argument error. exiting"
        return -1

    strNumber = sys.argv[1]
    number = int(strNumber)
    if number == 1:
        interfaces = "-i 1"
    elif number == 2:
        interfaces = "-i 1 -i 2"
    elif number == 3:
        interfaces = "-i 1 -i 2 -i 3"
    elif number == 4:
        interfaces = "-i 1 -i 2 -i 3 -i 4"

    os.system("sudo dumpcap -f \"udp port 4433 || icmp\" " + interfaces + " -w ./latest_wireshark_trace &")
    os.system("go run main.go -c -addr=10.0.1.4:4433")
    os.system("sudo killall dumpcap")

    return 0


if __name__ == "__main__":
    main()
