console.log("Project Dispatch")
console.log("Copyright 2017 Innovate Technologies")
console.log("====================================")

import os from "os"
import * as tagsUtil from "./components/utils/tags"
import * as machine from "./components/machine/machine"
import * as supervisor from "./components/machine/supervisor"

global.config = {
    bindPath: process.env.BINDPATH || "/var/run/dispatch",
    etcdAddress: process.env.ETCDADDR || "http://127.0.0.1:2379",
    publicIP: process.env.PUBLICIP || "127.0.0.1",
    machineName: process.env.MACHINENAME || os.hostname(),
    tags: tagsUtil.parseTags(process.env.TAGS || ""),
    arch: os.arch(),
}

console.log(`Starting Dispatch for ${config.machineName} (${config.arch}) on ${config.publicIP}`)
machine.init()
supervisor.init()
