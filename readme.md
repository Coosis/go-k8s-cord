server and agents hold endpoints

1. server and agents holds the same text-secret, keep it secret
2. request have to be signed with the secret

1. openssl genrsa -out ca.key 4096
2. openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt

3. openssl genrsa -out server.key 4096
4. openssl req -new -key server.key -out server.csr
5. openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 3650 -sha256

6. openssl genrsa -out agent.key 4096
7. openssl req -new -key agent.key -out agent.csr
8. openssl x509 -req -in agent.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out agent.crt -days 3650 -sha256
