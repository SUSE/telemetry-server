#!/bin/bash -eu

export PATH=/usr/bin:/bin:/usr/sbin:/sbin

tuser=tsvc
_localstatedir=/var
_tsvcdir=${_localstatedir}/lib/${tuser}
_certdir=${_tsvcdir}/certs
_defaults=/etc/default/susetelemetry

# load defaults if found
if [[ -s ${_defaults} ]]; then
	. ${_defaults}
fi

# determine which service to run and ensure binary exists
name=${TELEMETRY_SERVICE:-telemetry-server}
svc_path=/usr/bin/${name}
if [[ ! -x "${svc_path}" ]]; then
	echo "ERROR: Specified service binary '${svc_path}' doesn't exist."
	exit 1
fi

# ensure any additional certs are copied to the appropriate location
if [[ -d ${_certdir} ]]; then
	cp -a ${_certdir}/* /etc/pki/trust/anchors
fi

# ensure certs are up-to-date
update-ca-certificates

if [[ ! -d ${_tsvcdir} ]]; then
    echo "Error: no ${_tsvcdir} directory found; did you bind mount the service account dir to ${_tsvcdir}?"
    exit 1
fi

uid_gid=(
    $(stat -c '%u %g' ${_tsvcdir})
)

# ensure that the required group exists or add it with matching gid
# if needed
getent group ${tuser} >/dev/null || groupadd \
    -r \
    -g ${uid_gid[1]} \
    ${tuser}

# ensure that the required user exists or add it with matching uid
# if needed
getent passwd ${tuser} >/dev/null || useradd \
    -r \
    -g ${tuser} \
    -u ${uid_gid[0]} \
    -d ${_tsvcdir} \
    -s /sbin/nologin \
    -c "user for ${name}" ${tuser}

chown -R ${uid_gid[0]}:${uid_gid[1]} ${_tsvcdir}

# construct the command to be executed
cmd_args=(
    "${svc_path}"
    "${@}"
)

# add the --debug flag to args if log level is debug
case "${LOG_LEVEL:-info}" in
	(debug)
		cmd_args+=(--debug)
		;;
esac

# exec runuser to ensure signals are propagated to command
exec runuser -u ${tuser} -- "${cmd_args[@]}"
