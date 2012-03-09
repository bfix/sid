#!/bin/bash

openssl genrsa -out key.pem 2048
openssl req -new -key key.pem -out certreq.pem
openssl x509 -req -days 365 -in certreq.pem -signkey key.pem -out cert.pem

