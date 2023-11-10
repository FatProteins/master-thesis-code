/bin/envsubst < /thesis/config/fault-config-"${INSTANCE_NUMBER}".yml.tpl > /thesis/config/fault-config.yml
cd /thesis/bftsmart/build/install/library || exit
echo "Instance ID: ${INSTANCE_ID}"
bash smartrun.sh bftsmart.demo.map.MapServer "${INSTANCE_ID}"