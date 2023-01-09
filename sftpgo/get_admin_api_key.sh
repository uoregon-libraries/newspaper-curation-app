#!/usr/bin/env bash

# Get SFTPgo admin API key and store in NCA settings file

if [[ ! $(command -v jq) ]]; then
  echo "jq required. Please install jq"
  exit 1
fi

if [[ ! (-r ${SETTINGS_PATH} && -w ${SETTINGS_PATH}) ]]; then
  echo "Can't read and write file at SETTINGS_PATH value: '${SETTINGS_PATH}'"
  ls -l ${SETTINGS_PATH}
  exit 1
fi

source ${SETTINGS_PATH}

if [[ ${1:-} != '--force' ]]; then
  if [[ ! $(grep "!sftpgo_admin_api_key!" ${SETTINGS_PATH}) ]]; then
    echo "SFTPgo admin API key already set in NCA settings"
    $(dirname $0)/view_api_keys.sh
    exit 0
  fi
fi

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

RESPONSE=$(curl -s --show-error -X PUT \
  ${SFTPGO_API_URL}/${SFTPGO_ADMIN_LOGIN}/profile \
  -H "Authorization: Bearer ${TOKEN}" -d' { "allow_api_key_auth": true }')
if [[ $? != 0 ]]; then
  echo "Curl error with SFTPgo API request to update admin profile to allow API key auth"
  exit 1
elif [[ $(echo ${RESPONSE} | jq ".error") != "null" ]]; then
  echo "Error in response from SFTPgo API request to update admin profile to allow API key auth"
  echo ${RESPONSE} | jq
  exit 1
fi

RESPONSE=$(curl -s --show-error -X POST ${SFTPGO_API_URL}/apikeys \
  -H "Authorization: Bearer ${TOKEN}" \
  -d" {
    \"id\": \"001\",
    \"name\": \"nca_admin\",
    \"key\": \"key_pa55PHRA%E\",
    \"scope\": 1,
    \"description\": \"NCA Admin API Key\",
    \"admin\": \"${SFTPGO_ADMIN_LOGIN}\"
  }
")
if [[ $? != 0 ]]; then
  echo "Curl error with SFTPgo API request to create admin API key"
  exit 1
elif [[ $(echo ${RESPONSE} | jq ".error") != "null" ]]; then
  echo "Error in response from SFTPgo API request to create admin API key"
  echo ${RESPONSE} | jq
  exit 1
fi
APIKEY=$(
  echo ${RESPONSE} \
  | jq ".key" \
  | sed 's/^"\(.*\)"$/\1/'
)

echo "Ensuring global access to NCA settings file disabled:"
echo "chmod o-rwx ${SETTINGS_PATH}"
chmod o-rwx ${SETTINGS_PATH}

echo "Writing SFTPgo admin API key to NCA settings file: ${SETTINGS_PATH}"
# Copy approach needed to work with mounting settings file via Docker
sed "s/^SFTPGO_ADMIN_API_KEY=.*$/SFTPGO_ADMIN_API_KEY=${APIKEY}/" ${SETTINGS_PATH} > /tmp/settings-updated
cp /tmp/settings-updated ${SETTINGS_PATH}
rm /tmp/settings-updated

echo "Testing API key by requesting to view server status"
$(dirname $0)/test_api_key.sh

