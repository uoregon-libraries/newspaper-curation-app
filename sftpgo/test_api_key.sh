#!/usr/bin/env bash

# Test SFTPgo admin API key from settings to view server status

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

RESPONSE=$(curl -s --show-error -X GET ${SFTPGO_API_URL}/status \
  -H "X-SFTPGO-API-KEY: ${SFTPGO_ADMIN_API_KEY}")
if [[ $? != 0 ]]; then
  echo "Curl error with SFTPgo API request to view server status"
  exit 1
elif [[ $(echo ${RESPONSE} | jq ".error") != "null" ]]; then
  echo "Error in response from SFTPgo API request to view server status"
  echo ${RESPONSE} | jq
  exit 1
fi

echo "Server status:"
echo "${RESPONSE}" | jq

