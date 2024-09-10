import { getToken, loadGPX, loadActivity } from './api.js'
import { resizePane } from './resize.js'

async function create() {
    console.debug("Creating the page")
    let detail = {
        container: 'detail',
        splitsContainer: 'splits',
        data: {}
    }

    mapboxgl.accessToken = await getToken()
    console.debug("Token loaded")

    loadGPX().then(gpxText => {
        const map = createMap(gpxText)
        resizePane(map)
        console.debug("GPX loaded")
    }).catch(error => console.error("Error loading GPX", error))

    loadActivity().then(activity => {
        detail.data = parseData(activity)
        updateDetail(detail)
        showSplits(detail)
        console.debug("Activity loaded")
    }).catch(error => console.error("Error loading activity", error))

}

function createMap(gpxText) {
    const map = new mapboxgl.Map({
        container: 'map',
        style: 'mapbox://styles/mapbox/outdoors-v11',
        center: [10, 51],  // Default center
        zoom: 9
    });
    const parser = new DOMParser()
    const gpxDoc = parser.parseFromString(gpxText, 'application/xml')
    const geojson = toGeoJSON.gpx(gpxDoc)

    map.on('load', () => {
        map.addSource('gpxRoute', {
            type: 'geojson',
            data: geojson
        })

        map.setCenter(geojson.features[0].geometry.coordinates[0])
        map.setZoom(9)

        map.addLayer({
            id: 'gpxRouteLayer',
            type: 'line',
            source: 'gpxRoute',
            layout: {
                'line-join': 'round',
                'line-cap': 'round'
            },
            paint: {
                'line-color': '#FF5733',  // Change the color of the GPX route
                'line-width': 4
            }
        });

        // Fit the map to the GPX bounds
        const bounds = new mapboxgl.LngLatBounds();
        geojson.features.forEach(function(feature) {
            feature.geometry.coordinates.forEach(function(coord) {
                bounds.extend(coord);
            });
        });
        map.fitBounds(bounds);
    })

    return map
}

function updateDetail(detail) {
    const detailElement = document.getElementById(detail.container)
    detailElement.innerHTML = `
        <div class="card">
            <div class="card-header">
                <p class="card-text">
                   ${detail.data.Created.time} on ${detail.data.Created.date}
                </p>
                <h3 class="card-title">${detail.data.Name}</h5>
            </div>
            <div class="card-body">
                <ul class="list-group">
                    <li class="list-group-item">
                        <span>Distance</span><span>${detail.data.Distance}</span>
                    </li>
                    <li class="list-group-item">
                        <span>Elapsed Time</span>
                        <span>${detail.data.Duration}</span>
                    </li>
                    <li class="list-group-item">
                        <span>Average Pace</span>
                        <span>${detail.data.AveragePace}</span>
                    </li>
                    <li class="list-group-item">
                        <span>Elevation Gain:</span><span>${detail.data.Uphill}</span>
                    </li>
                    <li class="list-group-item">
                        <span>Starting Elevation:</span><span>${detail.data.Elevation.toFixed(0)}m</span>
                    </li>
                    <li class="list-group-item">
                        <span>Highest Point:</span><span>${parseInt(detail.data.Elevation.toFixed(0)) + parseInt(detail.data.Uphill)}m</span>
                    </li>
                </ul>
            </div>
        </div>
    `
}

function showSplits(detail) {
    let splitsElement = document.getElementById(detail.splitsContainer)

    let content = `
    <div class="card">
            <div class="card-header">
                <h3 class="card-title">Splits</h3>
            </div>
            <ul class="list-group">
                <li class="list-header list-group-item">
                    <span class="km-header">KM</span>
                    <span class="pace-header">Pace</span>
                    <span class="elev-header">Elev</span>
                </li>
        `

    detail.data.Splits.forEach((item, idx) => {
        let minsPerKm = item.SplitTime / 60
        let elevation = item.Elevation - detail.data.Splits[idx - 1]?.Elevation || item.Elevation - detail.data.Elevation
        // display min per km as 00:00/km
        minsPerKm = minsPerKm.toFixed(0) + ":" + String(Math.round((minsPerKm % 1) * 60)).padStart(2, '0')
        content += `
        <li class="list-group-item">
            <span class="km">${(item.Distance / 1000) < 1 ? (item.Distance / 1000).toFixed(2) : parseInt((item.Distance / 1000).toFixed(0)) + parseInt(idx)}</span>
            <span class="pace">${minsPerKm} /km</span>
            <span class="elev">${elevation.toFixed(0)} m<span>
        </li>`
    })

    splitsElement.innerHTML = content + "</ul></div>"
}

function parseData(gpxJSON) {
    let created = {
        date: '',
        time: ''
    }

    const d = new Date(gpxJSON.CompletedDate)
    created.date = d.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        weekday: 'long'
    })
    created.time = d.toLocaleTimeString('en-US', {
        hour: 'numeric',
        minute: '2-digit'
    })

    // Extract the whole minutes
    const minutes = Math.floor(gpxJSON.AveragePace);

    // Extract the seconds by taking the decimal part and converting it to seconds
    const seconds = Math.round((gpxJSON.AveragePace - minutes) * 60);

    // Ensure seconds are always two digits
    const formattedSeconds = String(seconds).padStart(2, '0');

    // Return formatted string "MM:SS"
    const avgpace = `${minutes}:${formattedSeconds}/km`;

    return {
        Name: gpxJSON.Name,
        Created: created,
        Distance: (gpxJSON.Distance / 1000).toFixed(2) + "km",
        Uphill: gpxJSON.Uphill.toFixed(0) + "m",
        Downhill: gpxJSON.Downhill.toFixed(0) + "m",
        Duration: secondsToHMS(gpxJSON.Time),
        AveragePace: avgpace,
        Elevation: gpxJSON.Elevation,
        Splits: gpxJSON.Splits
    }
}

function secondsToHMS(seconds) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    return (
        String(hours).padStart(2, '0') + ':' +
        String(minutes).padStart(2, '0') + ':' +
        String(secs).padStart(2, '0')
    );
}

window.onload = create()
