export const parseTags = (tagString) => {
    const tagsArray = tagString.split(",")
    const tags = {}

    for (let tag of tagsArray) {
        const tagInfo = tag.split("=")
        if (tagInfo.length === 2) {
            tags[tagInfo[0]] = tagInfo[0]
        }
    }

    return tags
}
