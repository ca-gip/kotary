#!/bin/bash

#
# A FEW INFORMATIONS ABOUT THIS SCRIPT
# 
# This script is used to test the kotary operator, it replicates what a really 
# simple a basic usage of kotary should look like. 
# This script is not an exaustive list of test of the operator, it is closer to 
# an end to end test because each test depends of the previous one and fails one the first error encountered.
# 
# This test should be used to verify that every basic features of kotary is working fine.
# It is not meant to be used to debug a particular part of the operator.
#
# HOW THIS SCRIPT WORKS:
#
# First of all it deploys the crds nececary for the operator and then deploys it in you current cluster.
# (if you are using kind don't forget to load you image in the cluster)
# Then it goes into a series of tests described in the code below.
# If any error occurs or on any unwanted behavior, the script ends and starts the CleanUp function
# to remoove what have been used during the test. 
# Note that you can uncomment some lines in the CleanUp function depending of your needs.
# If everything goes as intended the script will exit with a code 0 and cleanup the evironment.
#  
# /!\ This script is in no way perfect, feel free to add new tests at the end of the script if you
# believe that the script needs some more coverage.
#
# Pre-requirements to run this script:
#   - having kubectl installed
#   - beeing connected to a dev cluster
#   - having jq installed
#
# @author: Léo ARPIN (ty for reading :D )
#

set -e

# Bash colors for a more enjoyable script
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color



NS=$(cat /dev/urandom | tr -cd 'a-f0-9' | head -c 12) #generate random namespace to avoid conflicts
ROOT=$(git rev-parse --show-toplevel)   #get rootpath of git repo
QUOTACLAIM='resourcequotaclaims.cagip.github.com'

# Clean up function. If you want to totaly remove kotary after test, uncomment the lines
CleanUp () {
    echo -e "\\n${BLUE}Starting CleanUp ${NC}\\n"
    kubectl delete ns $NS
    #kubectl delete configmap -n kube-system kotary-config
    #kubectl delete deployment -n kube-system kotary
    #kubectl delete crd resourcequotaclaims.cagip.github.com
    rm temp.json
}

trap CleanUp EXIT ERR

echo -e "${BLUE}====== Starting SetUp ======${NC} \\n"

#apply crds, if an error occured it might be that the user is not connected to a cluster
if ! kubectl apply -f $ROOT/artifacts/crd.yml ;
 then echo -e "\\n${RED}CONNECT TO A CLUSTER BEFORE RUNNING THIS EXECUTABLE${NC}\\n" && exit 1 ; fi

#deploy the operator
kubectl apply -f $ROOT/artifacts/deployment.yml
kubectl apply -f $ROOT/e2e/KotaryService/ConfigMap.yaml -n kube-system

kubectl create ns $NS
while ! kubectl get pods -n kube-system | grep kotary | grep Running > /dev/null ; do echo -e "${BLUE}.... Waiting for Kotary pod to be Running ....${NC}" ; sleep 2; done

#This is the test part
echo -e "\\n${BLUE}====== Starting Tests ======${NC}\\n"

#Trying to apply a rqc and verify that the claim is accepted (an accepted claim is deleted from the queue so it does not return anything) 
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaim.yaml -n $NS && sleep 3
kubectl get $QUOTACLAIM -n $NS -o=json > temp.json #get the claim 
phase=$(jq ' .items[].status.phase' temp.json) #get the status of the claim if the claim has been accepted $phase will be empty
if [  "$phase" != "" ]; #if the phase isn't empty, then it is an error
 then echo -e "\\n${RED}FAILLED! error durring Claim test: the Claim is $phase. Should be accepted ${NC}" && exit 1 ; fi

#apply pods in the NS in order to fill both cpu and memory resources
#if you add every spec of each pod you end up at cpu: 500m/660m, memory: 1Gi/1Gi
#if the result is different it should be considered an error
echo -e "\\n ${PURPLE}-- Applying pods in NS --${NC}" && sleep 3
kubectl apply -f $ROOT/e2e/KindConfig/pod1.yml -n $NS
kubectl apply -f $ROOT/e2e/KindConfig/pod2.yml -n $NS
kubectl apply -f $ROOT/e2e/KindConfig/pod3.yml -n $NS
kubectl apply -f $ROOT/e2e/KindConfig/pod4.yml -n $NS
echo -e "\\n ${PURPLE}Should be 'cpu: 500m/660m, memory: 1000Mi/1Gi'${NC}"
if ! kubectl get resourcequota -n $NS | grep "cpu: 500m/660m, memory: 1000Mi/1Gi";
  then echo -e "\\n${RED}FAILLED! Error, the expected specs are not the same as the actual ones.${NC}" && exit 1  ; fi
echo -e "${GREEN} -- OK --${NC}\\n"

# Verify that trying to add a pod with resources exceeding what is left to use results in an error
echo -e "\\n ${PURPLE}-- Trying to add a pod over max ressources (must be forbidden) --${NC}" && sleep 3
if kubectl apply -f $ROOT/e2e/KindConfig/pod5.yml -n $NS ; # if the command does NOT result in an error then the test fails
 then echo -e "\\n${RED}FAILLED! error durring Pod test: The pod must not be accepted because it uses more ressources than what's left to use.${NC}" && exit 1  ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"

# Apply a new quotaclaim to scale up the resourses
# verify that the claim is accepted (nothing should appear in the 'status' field)
echo -e "\\n ${PURPLE}-- Scale UP --${NC}"
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaimUp.yaml -n $NS && sleep 3 #apply the new rqc
kubectl get $QUOTACLAIM -n $NS -o=json > temp.json #get the claim 
phase=$(jq ' .items[].status.phase' temp.json) #get the status of the claim if the claim has been accepted $phase will be empty
if [  "$phase" != "" ]; #if the phase isn't empty, then it is an error
 then  echo -e "\\n${RED}FAILLED! error durring Scale UP: the Claim is $phase ${NC}\\n" && kubectl get $QUOTACLAIM -n $NS  && exit 1 ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"

# Apply a new quotaclaim to scale up the resourses but this claim is to big,
# Kotary should see that what is requested is way to much and should reject the claim.
# assert that the rqc is rejected
echo -e "\\n ${PURPLE}-- Scale UP(to big) --${NC}"
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaimToBig.yaml -n $NS && sleep 3
kubectl get $QUOTACLAIM -n $NS -o=json > temp.json
phase=$(jq ' .items[].status.phase' temp.json)
if [  "$phase" != "\"REJECTED\"" ]; #The claim MUST be rejected, else it is an error
 then echo -e "\\n${RED}FAILLED! error durring Scale UP(to big): the Claim has not been rejected${NC}" && kubectl get $QUOTACLAIM -n $NS  && exit 1 ; fi
 echo -e "${GREEN} -- OK --${NC}\\n" && sleep 3

# Apply a new quotaclaim to scale down the resourses,
# Kotary should see that what is requested is lower that what is curently used.
# assert that the rqc is set to pending
echo -e "\\n ${PURPLE}-- Scale Down (under what is curently used --> PENDING) --${NC}"
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaimPending.yaml -n $NS && sleep 3
kubectl get $QUOTACLAIM -n $NS -o=json > temp.json
phase=$(jq ' .items[].status.phase' temp.json)
if [  "$phase" != "\"PENDING\"" ]; #The claim MUST be pending, else it is an error
 then echo -e "\\n${RED}FAILLED! error durring pending test: the Claim is not set to PENDING${NC}" && kubectl get $QUOTACLAIM -n $NS && exit 1 ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"

# Reduce the current usage of cpu and memory by deleting a pod
echo -e "\\n ${PURPLE}-- Delete pod-4: the pending claim should now be accepted --${NC}" && sleep 3
kubectl delete pod -n $NS podtest-4 && sleep 3

# assert that, after deletion of the pod, the 'pending' claim is now accepted
kubectl get $QUOTACLAIM -n $NS -o=json > temp.json
phase=$(jq ' .items[].status.phase' temp.json)
if [  "$phase" != "" ]; #The status must be empty because the claim should now be accepted. (remember: empty=accepted)
 then echo -e "\\n${RED}FAILLED! error durring pending test: the PENDING Claim is not accepted after resources are updated${NC}" && kubectl get $QUOTACLAIM -n $NS && exit 1; fi
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaim.yaml -n $NS  && sleep 3
echo -e "${GREEN} -- OK --${NC}\\n"

echo -e "\\n${GREEN} <<< ALL GOOD, Well done! :) >>>${NC}"