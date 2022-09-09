get_token() {
  curl --basic http://admin:password@localhost/sftpgo/api/v2/token | \
    jq ".access_token" | \
    sed 's|^"\(.*\)"$|\1|'
}

curl --basic http://localhost/sftpgo/api/v2/version -H "Authorization: Bearer $(get_token)"
