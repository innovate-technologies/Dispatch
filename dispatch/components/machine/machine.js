import os from "os"
import * as etcd from "../etcd/etcd.js"

export const init = async () => {
    const machineKey = `/dispatch/machines/${config.machineName}/`
    await etcd.deleteKey(machineKey, { recursive: true })

    await etcd.setKey(`${machineKey}/ip`, config.publicIP)
    await etcd.setKey(`${machineKey}/arch`, config.arch)
    for (let name in config.tags) {
        if (config.tags.hasOwnProperty(name)) {
            await etcd.setKey(`${machineKey}/tags/${name}`, config.tags[name])
        }
    }
    await etcd.setKey(`/dispatch/machines/${config.machineName}/alive`, 1, 10)
    setInterval(aliveInterval, 1000)
}

const aliveInterval = async () => {
    await etcd.refreshKey(`/dispatch/machines/${config.machineName}/alive`, 10)
    await etcd.setKey(`/dispatch/machines/${config.machineName}/load`, os.loadavg().toString())
}
