#!/bin/bash
# (C) Copyright [2020] Hewlett Packard Enterprise Development LP
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.

declare PID=0
declare OWN_PID=$$

sigterm_handler()
{
        if [[ $PID -ne 0 ]]; then
                sleep 1
                kill -9 $PID
                wait "$PID" 2>/dev/null
        fi
        exit 0
}

# create a signal trap
create_signal_trap()
{
        trap 'echo "[$(date)] -- INFO  -- SIGTERM received for BMC Operator, initiating shut down"; sigterm_handler' SIGTERM
}

# keep the script running till SIGTERM is received
run_forever()
{
        wait
}

start_bmc_operator()
{ 
	nohup /manager >> /var/log/operator_logs/bmc_operator.log 2>&1 &
	PID=$!
	sleep 3

}

monitor_process()
{
        while true; do
                pid=$(pgrep -fc manager 2> /dev/null)
                if [[ $? -ne 0 ]] || [[ $pid -gt 1 ]]; then
                        echo "[$(date)] -- ERROR -- BMC Operator not found running, exiting"
			kill -15 ${OWN_PID}
                        exit 1
                fi
                sleep 5
        done &
}

##############################################
###############  MAIN  #######################
##############################################

start_bmc_operator

create_signal_trap

monitor_process

run_forever

exit 0
