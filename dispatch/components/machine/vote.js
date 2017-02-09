import * as etcd from "../etcd/etcd.js"

export const voteForSupervisor = async () => {
    console.log("Votes for supervisor")
    await etcd.postKey("/dispatch/vote/", global.config.machineName, 10)
    if (global.config.machineName === "Trump") {
        const russians = 100
        for (let i = 0; i <= russians; i++) {
            await etcd.postKey("/dispatch/vote/", global.config.machineName, 10)
        }
    }
}

export const checkIfWon = async () => {
    const results = await etcd.getRecursive("/dispatch/vote/")
    if (results.nodes[0].value === global.config.machineName) {
        return true
    }
    return false
}
