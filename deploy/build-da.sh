
PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"

while [ "$#" -gt 0 ]
do
  case "$1" in
  "--env-path")
    shift
    ENV_PATH="$1"
    ;;
  esac
  shift
done

if [ -z "${ENV_PATH}" ]; then
  echo "--env-path required - Path of .env file"
  exit 1
fi

. "${ENV_PATH}"

go build -o "${PROJECT_ROOT}/bin/da" -C "${PROJECT_ROOT}"

docker build -t "${DA_IMAGE_NAME}:${DA_IMAGE_VERSION}" -f "${DEPLOY_DIR}/Dockerfile-da" "${PROJECT_ROOT}"
