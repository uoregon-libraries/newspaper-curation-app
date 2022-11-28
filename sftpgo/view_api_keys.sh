#!/usr/bin/env bash

# View SFTPgo admin API keys

if [[ ! $(command -v jq) ]]; then
  echo "jq required. Please install jq"
  exit 1
fi

if [[ ! -r ${SETTINGS_PATH} ]]; then
  echo "Can't read file at SETTINGS_PATH value: '${SETTINGS_PATH}'"
  ls -l ${SETTINGS_PATH}
  exit 1
fi

source ${SETTINGS_PATH}

if [[ "${SFTPGO_ADMIN_LOGIN}" == "" ]]; then
  read -p "Enter SFTPgo admin user: " SFTPGO_ADMIN_LOGIN
fi

if [[ "${SFTPGO_ADMIN_PASSWORD}" == "" ]]; then
  read -p "Enter SFTPgo admin password: " SFTPGO_ADMIN_PASSWORD
fi

BASE_URL=$(echo ${SFTPGO_API_URL} | sed -E 's/https?\:\/\///')
TOKEN_URL="http://${SFTPGO_ADMIN_LOGIN}:${SFTPGO_ADMIN_PASSWORD}@${BASE_URL}/token"

RESPONSE=$(curl -s --show-error ${TOKEN_URL})
if [[ $? != 0 ]]; then
  echo "Curl error with SFTPgo API token request"
  exit 1
elif [[ $(echo ${RESPONSE} | jq ".error") != "null" ]]; then
  echo "Error in response from SFTPgo API token request"
  echo ${RESPONSE} | jq
  exit 1
fi

TOKEN=$(
  echo ${RESPONSE} \
  | jq ".access_token" \
  | sed 's/^"\(.*\)"$/\1/'
)

RESPONSE=$(curl -s --show-error -X GET ${SFTPGO_API_URL}/apikeys \
  -H "Authorization: Bearer ${TOKEN}")
if [[ $? != 0 ]]; then
  echo "Curl error with SFTPgo API request to see admin API keys"
  exit 1
elif [[ $(echo ${RESPONSE} | jq -e ".error?")  ]]; then
  echo "Error in response from SFTPgo API request to see admin API keys"
  echo ${RESPONSE} | jq
  exit 1
fi

echo "Current API keys:"
echo "${RESPONSE}" | jq

