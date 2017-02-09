import * as etcd from "../etcd/etcd.js"
import * as vote from "./vote"

export const init = async () => {
    try {
        await etcd.getKey("/dispatch/supervisor/alive")
    } catch (error) {
        // no supervisor! Let's become one
        await vote.voteForSupervisor()
        if (await vote.checkIfWon()) {
            console.log("become supervisor")
            await becomeSupervisor()
        }
    }
    watchForSupervisorToDie()
}

const watchForSupervisorToDie = async () => {
    const key = await etcd.watchKey("/dispatch/supervisor/alive")
    if (!key.value) {
        await vote.voteForSupervisor()
        if (await vote.checkIfWon()) {
            console.log("become supervisor")
            return await becomeSupervisor()
        }
    }
    return watchForSupervisorToDie()
}

const becomeSupervisor = async () => {
    await etcd.setKey("/dispatch/supervisor/alive", 1, 10)
    await etcd.setKey("/dispatch/supervisor/machine", global.config.machineName, 10)
    setInterval(makeSureThePesantsKnow, 1000)
    setInterval(doDuty, 1000)
}

const makeSureThePesantsKnow = async () => {
    await etcd.refreshKey("/dispatch/supervisor/alive", 10)
    await etcd.refreshKey("/dispatch/supervisor/machine", 10)
}

const doDuty = async () => {
    const machines = (await etcd.getDirectory("/dispatch/machines")).nodes
    for (let machine of machines) {
        try {
            await etcd.getKey(`${machine.key}/alive`)
        } catch (error) {
            await etcd.deleteKey(machine.key, { recursive: true })
        }
    }
}
