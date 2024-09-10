async function getToken() {
    console.log("getting token")
    try {
        const response = await fetch("http://localhost:8222/token")
        if (!response.ok) {
            throw new Error("Error getting token")
        }

        const token = await response.json()
        return token.token
    } catch (error) {
        console.error("Error getting token", error)
    }
}

async function loadGPX(gpxID) {
    if (!gpxID) {
        gpxID = 1
    }
    const gpxURL = 'http://localhost:8222/gpx/' + gpxID

    try {
        const response = await fetch(gpxURL)
        if (!response.ok) {
            throw new Error("Getting GPX failed due to server error");
        }

        return await response.text()
    } catch (error) {
        console.error("Getting GPX failed due to server error", error)
    }
}


async function loadActivity(gpxID) {
    if (!gpxID) {
        gpxID = 1
    }
    const gpxURL = "http://localhost:8222/gpx/" + gpxID + "/detail"
    try {
        const response = await fetch(gpxURL)
        if (!response.ok) {
            throw new Error("Error getting activity details")
        }

        return await response.json()
    } catch (error) {
        console.error("Error getting activity details", error)
    }
}

export { getToken, loadGPX, loadActivity };
