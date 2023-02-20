#!/bin/bash
BUILD_DIR=build/
UKVM_BIN=${INCLUDEOS_PREFIX}/includeos/x86_64/lib/ukvm-bin
TAP_IF=tap100
SOCKET_IF=/tmp/solo5_socket
MIG_FILE=mig_file

function usage {
  cat <<EOF

Usage: 
  $(basename ${0}) [<options>]

Options: 
  -h                    print help

  -g                    run the unikernel in debug mode (gdb)

  -m                    enable VM migration

  -l                    load VM from migration file

  -b <build_dir>        set path to dir including binary 
                        (default: ${BUILD_DIR})

  -u <ukvm-bin>         set path to ukvm-bin 
                        (default: ${UKVM_BIN})

  -n <network device>   set tap device 
                        (default: ${TAP_IF})

  -s <socket name>      set socket file name 
                        (default: ${SOCKET_IF})

  -f <migfile name>     set migration file name
                        (default: ${MIG_FILE})

  -a <arguments...>     set arguments for the app 

EOF
}


########### get arguments #############
GDB_FLAG=false
MON_FLAG=false
LOAD_FLAG=false

while getopts hgmlb:u:n:s:f:a: OPT
do
  case $OPT in
    "h" )
      usage
      exit -1 ;;
    "g" )
      GDB_FLAG=true ;;
    "m" )
      MON_FLAG=true ;;
    "l" )
      LOAD_FLAG=true ;;
    "b" )
      USER_BUILD_DIR=${OPTARG} ;;
    "u" )
      USER_UKVM_BIN=${OPTARG} ;;
    "n" )
      USER_TAP_IF=${OPTARG} ;;
    "s" )
      USER_SOCKET_IF=${OPTARG} ;;
    "f" )
      USER_MIG_FILE=${OPTARG} ;;
    "a" )
      USER_ARGS=${OPTARG} ;;
  esac
done

########### set arguments #############
if [ ! -z ${USER_BUILD_DIR} ]; then
  echo "INFO: ${USER_BUILD_DIR} is specified as the build directory."
  BUILD_DIR=${USER_BUILD_DIR}
fi

if [ ! -z ${USER_UKVM_BIN} ]; then
  echo "INFO: ${USER_UKVM_BIN} is used as the backend monitor."
  UKVM_BIN=${USER_UKVM_BIN}
fi

if [ ! -z ${USER_SOCKET_IF} ]; then
  echo "INFO: ${USER_SOCKET_IF} is used as a socket interface."
  SOCKET_IF=${USER_SOCKET_IF}
fi

if [ ! -z ${USER_MIG_FILE} ]; then
  echo "INFO: ${USER_MIG_FILE} is used as a socket interface."
  MIG_FILE=${USER_MIG_FILE}
fi

if [ ! -z ${USER_TAP_IF} ]; then
  echo "INFO: ${USER_TAP_IF} is used as a tap interface."
  TAP_IF=${USER_TAP_IF}
fi

########### Error check #############
if [ ! -e ${BUILD_DIR} ]; then
  echo "Error: ${BUILD_DIR} doesn't exist. Please specify the correct build directory or compile an application first."
  exit -1
fi

# search build_dir for unikernel app binary
APP_BIN=$(find ${BUILD_DIR} -maxdepth 1 -executable -type f)

if [ ! -e ${APP_BIN} ]; then
  echo "Error: binary ${APP_BIN} is missing. Please specify the correct build directory."
  exit -1
fi

if [ -z ${XILINX_XRT} ]; then
  echo "Error: XILINX_XRT is not set. Please install XRT."
  exit -1
fi

if [ -z ${INCLUDEOS_PREFIX} ]; then
  echo "Error: INCLUDEOS_PREFIX is not set. Please install Funky OS."
  exit -1
fi


########### Execute app #############
# apply exec permission to ukvm bin
if [ -e ${UKVM_BIN} -a ! -x ${UKVM_BIN} ]; then
  chmod a+x ${UKVM_BIN}
fi

if "${MON_FLAG}" ; then
  MON_OPT="--mon=${SOCKET_IF}"
fi

if "${LOAD_FLAG}" ; then
  LOAD_OPT="--load=${MIG_FILE}"
fi

if "${GDB_FLAG}" ; then
  echo "Usage: run --mem=1024 --disk=${APP_BIN} --net=${TAP_IF} ${MON_OPT} ${LOAD_OPT} ${APP_BIN} ${USER_ARGS}"
  echo "Press the Enter to start gdb..."
  read Wait
  gdb -tui ${UKVM_BIN} 
else 
  echo "${UKVM_BIN} --mem=1024 --disk=${APP_BIN} --net=${TAP_IF} ${MON_OPT} ${LOAD_OPT} ${APP_BIN} ${USER_ARGS}"
  ${UKVM_BIN} --mem=1024 --disk=${APP_BIN} --net=${TAP_IF} ${MON_OPT} ${LOAD_OPT} ${APP_BIN} ${USER_ARGS}
fi
