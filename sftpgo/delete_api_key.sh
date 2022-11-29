#!/usr/bin/env bash

# Delete SFTPgo admin API key

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

if [[ "$1" == "" ]]; then
  read -p "Enter API key ID to delete: " API_KEY_ID
else
  API_KEY_ID=$1
fi

RESPONSE=$(curl -s --show-error -X DELETE ${SFTPGO_API_URL}/apikeys/${API_KEY_ID} \
  -H "Authorization: Bearer ${TOKEN}")
if [[ $? != 0 ]]; then
  echo "Curl error with SFTPgo API request to delete API key ID ${API_KEY_ID}"
  exit 1
elif [[ $(echo ${RESPONSE} | jq ".error") != "null" ]]; then
  echo "Error in response from SFTPgo API request to delete API key ID ${API_KEY_ID}"
  echo ${RESPONSE} | jq
  exit 1
fi

echo "API key ID ${API_KEY_ID} delete response:"
echo "${RESPONSE}" | jq

