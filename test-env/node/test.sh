#!/bin/bash
cd /testsetup
dispatchctl load-unit -g nginx.service
dispatchctl list-units
dispatchctl load-unit test1.service
dispatchctl list-units
dispatchctl load-unit test2.service
dispatchctl list-units
dispatchctl load-unit test3.service
dispatchctl list-units