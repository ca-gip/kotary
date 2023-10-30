#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

#YOU NEED TO BE INSIDE A K8S CLUSTER TO RUN THIS#

NS=$(cat /dev/urandom | tr -cd 'a-f0-9' | head -c 12) #generate random namespace
ROOT=$(git rev-parse --show-toplevel)   #get root of git repo

CleanUp () {
    echo -e "\\n${BLUE}Starting CleanUp ${NC}\\n"
    kubectl delete ns $NS
    #kubectl delete configmap -n kube-system kotary-config
    #kubectl delete deployment -n kube-system kotary
    #kubectl delete crd resourcequotaclaims.cagip.github.com
}

echo -e "${BLUE}====== Starting SetUp ======${NC} \\n"

if ! kubectl apply -f artifacts/crd.yml ;
 then echo -e "\\n${RED}CONNECT TO A CLUSTER BEFORE RUNNING THIS EXECUTABLE${NC}\\n" && exit 1 ; fi

kubectl apply -f artifacts/deployment.yml
kubectl -n kube-system create -f $ROOT/e2e/KotaryService/ConfigMap.yaml

kubectl create ns $NS
while  ! kubectl get pods -n kube-system | grep kotary | grep Running > /dev/null ; do echo -e "${BLUE}.... Waiting for Kotary pod to be Running ....${NC}" ; sleep 2; done

echo -e "\\n${BLUE}====== Starting Tests ======${NC}\\n"
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaim.yaml -n $NS
if kubectl get quotaclaim -n $NS | grep REJECTED ;
 then echo -e "\\n${RED}FAILLED! error durring Claim test: the Claim is REJECTED. Should be accepted ${NC}" && CleanUp && exit 1 ; fi
kubectl get resourcequota -n $NS

echo -e "\\n ${PURPLE}-- Applying pods in NS --${NC}" && sleep 1
kubectl apply -f $ROOT/e2e/KindConfig/pod1.yml -n $NS
kubectl apply -f $ROOT/e2e/KindConfig/pod2.yml -n $NS
kubectl apply -f $ROOT/e2e/KindConfig/pod3.yml -n $NS
kubectl apply -f $ROOT/e2e/KindConfig/pod4.yml -n $NS
echo -e "\\n ${PURPLE}Should be 'cpu: 500m/660m, memory: 1000Mi/1Gi'${NC}"
kubectl get resourcequota -n $NS
echo -e "${GREEN} -- OK --${NC}\\n"

echo -e "\\n ${PURPLE}-- Trying to add a pod over max ressources (must be forbidden) --${NC}" && sleep 1
if kubectl apply -f $ROOT/e2e/KindConfig/pod5.yml -n $NS ;
 then echo -e "\\n${RED}FAILLED! error durring Pod test: The pod must not be accepted because it uses more ressources than what's left to use.${NC}" && CleanUp && exit 1  ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"


echo -e "\\n ${PURPLE}-- Scale UP --${NC}" && sleep 1
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaimUp.yaml -n $NS
if kubectl get quotaclaim -n $NS | grep REJECTED ;
 then  echo -e "\\n${RED}FAILLED! error durring Scale UP: the Claim has been rejected${NC}\\n" && kubectl get quotaclaim -n $NS && CleanUp && exit 1 ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"

echo -e "\\n ${PURPLE}-- Scale UP(to big) --${NC}" && sleep 1
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaimToBig.yaml -n $NS
if ! kubectl get quotaclaim -n $NS | grep REJECTED ;
 then echo -e "\\n${RED}FAILLED! error durring Scale UP(to big): the Claim has not been rejected${NC}" && kubectl get quotaclaim -n $NS && CleanUp && exit 1 ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"


echo -e "\\n ${PURPLE}-- Scale Down (under what is curently used --> PENDING) --${NC}" && sleep 1
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaimPending.yaml -n $NS
if ! kubectl get quotaclaim -n $NS | grep PENDING ;
 then echo -e "\\n${RED}FAILLED! error durring pending test: the Claim is not set to PENDING${NC}" && kubectl get resourcequota -n $NS && CleanUp && exit 1 ; fi
 echo -e "${GREEN} -- OK --${NC}\\n"

echo -e "\\n ${PURPLE}-- Delete pod-4: the pending claim should now be accepted --${NC}" && sleep 1
kubectl delete pod -n $NS podtest-4 && sleep 1

if kubectl get quotaclaim -n $NS | grep PENDING ;
 then echo -e "\\n${RED}FAILLED! error durring pending test: the PENDING Claim is not accepted after resources are updated${NC}" && kubectl get quotaclaim -n $NS && CleanUp && exit 1; fi
kubectl apply -f $ROOT/e2e/KotaryService/QuotaClaim.yaml -n $NS
echo -e "${GREEN} -- OK --${NC}\\n"

echo -e "\\n ${PURPLE}-- Adding a pod with bad image --> should not impact the ressources used --${NC}" && sleep 1
kubectl apply -f $ROOT/e2e/KindConfig/badpod.yml -n $NS
if kubectl get resourcequota -n $NS | grep "350m/660m" && grep "750Mi/1Gi" ;
 then echo -e "\\n${RED}FAILLED! error durring resource test: Not RUNNING pod is not ignored when calculating the resourcequota${NC}" && CleanUp && exit 1; fi
echo -e "${GREEN} -- OK --${NC}\\n"


echo -e "\\n${GREEN} <<< ALL GOOD, Well done! :) >>>${NC}"

CleanUp

echo -e "\\n${BLUE}Done!${NC}"