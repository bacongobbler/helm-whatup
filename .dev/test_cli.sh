#!/bin/bash

runCov() {
    make cov
    mv c.out $(date +%s).out
}

mv c.out 1.out

## unset HELM_HOST and set TILLER_HOST
export TILLER_HOST=${HELM_HOST}
unset HELM_HOST

runCov

## reset Helm and Enable TLS
helm reset --force || (echo "Could not Reset Helm & Tiller" && exit 1)

(
    openssl genrsa -out ./ca.key.pem 4096
    openssl req -key ca.key.pem -new -x509 -days 7300 -sha256 -out ca.cert.pem -extensions v3_ca \
        -subj "/C=DE/ST=Bayern/L=Neu-Ulm/O=FABMation GmbH/OU=OpenSource Applications/CN=Helm Tiller CA"

    openssl genrsa -out ./tiller.key.pem 4096
    openssl genrsa -out ./helm.key.pem 4096
    openssl req -key tiller.key.pem -new -sha256 -out tiller.csr.pem \
        -subj "/C=DE/ST=Bayern/L=Neu-Ulm/O=FABMation GmbH/OU=OpenSource Applications/CN=tiller-server"
    openssl req -key helm.key.pem -new -sha256 -out helm.csr.pem \
        -subj "/C=DE/ST=Bayern/L=Neu-Ulm/O=FABMation GmbH/OU=OpenSource Applications/CN=helm-client"

    openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -CAcreateserial -in tiller.csr.pem -out tiller.cert.pem -days 365
    openssl x509 -req -CA ca.cert.pem -CAkey ca.key.pem -CAcreateserial -in helm.csr.pem -out helm.cert.pem  -days 365
)

# init Helm/ Tiller
helm init --tiller-tls \
    --tiller-tls-cert ./tiller.cert.pem \
    --tiller-tls-key ./tiller.key.pem \
    --tls-ca-cert ca.cert.pem \
    --tiller-tls-verify

cp ca.cert.pem $(helm home)/ca.pem
cp helm.cert.pem $(helm home)/cert.pem
cp helm.key.pem $(helm home)/key.pem

# configure Helm
export HELM_TLS_ENABLE=true
export HELM_TLS_VERIFY=false

runCov

#### >>>>>>>>>> Merge all Coverage Tests into one <<<<<<<<<< ####
go get github.com/wadey/gocovmerge
gocovmerge *.out > c.out