san-conf := """
[req]
distinguished_name = dn
req_extensions = v3_req
prompt = no

[dn]
CN = localhost

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
"""

san-conf:
	echo "{{san-conf}}" > san.conf

san-conf-domain domain: san-conf
	echo "DNS.1 = {{domain}}" >> san.conf

san-conf-ip ip: san-conf
	echo "IP.3 = {{ip}}" >> san.conf

clean-san-conf:
	rm -f san.conf

gen-ca:
	openssl genrsa -out ca.key 4096
	openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
		-out ca.crt

gen-central:
	just san-conf
	openssl genrsa -out server.key 4096
	openssl req -new -key server.key -out server.csr \
		-config san.conf
	openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
		-out server.crt -days 3650 -sha256 \
		-extfile san.conf -extensions v3_req
	just clean-san-conf

gen-agent:
	just san-conf
	openssl genrsa -out agent.key 4096
	openssl req -new -key agent.key -out agent.csr \
		-config san.conf
	openssl x509 -req -in agent.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
		-out agent.crt -days 3650 -sha256 \
		-extfile san.conf -extensions v3_req
	just clean-san-conf

gen-agent-domain domain:
	just san-conf-domain {{domain}}
	openssl genrsa -out agent.key 4096
	openssl req -new -key agent.key -out agent.csr \
		-config san.conf
	openssl x509 -req -in agent.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
		-out agent.crt -days 3650 -sha256 \
		-extfile san.conf -extensions v3_req
	just clean-san-conf

gen-agent-ip ip:
	just san-conf-ip {{ip}}
	openssl genrsa -out agent.key 4096
	openssl req -new -key agent.key -out agent.csr \
		-config san.conf
	openssl x509 -req -in agent.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
		-out agent.crt -days 3650 -sha256 \
		-extfile san.conf -extensions v3_req
	just clean-san-conf

gen-tool-cert:
	openssl genrsa -out cli.key 4096
	openssl req -new -key cli.key -out cli.csr
	openssl x509 -req -in cli.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
		-out cli.crt -days 3650 -sha256

gen-grpc:
	protoc --go_out=./internal/pb --go-grpc_out=./internal/pb \
		proto/v1/central.proto proto/v1/agent.proto

clean:
	rm -f ca.key ca.crt ca.srl
	rm -f server.key server.csr server.crt
	rm -f agent.key agent.csr agent.crt
	rm -f san.conf
