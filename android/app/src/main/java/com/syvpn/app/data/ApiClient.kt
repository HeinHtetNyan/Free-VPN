package com.syvpn.app.data

import org.json.JSONObject
import java.io.OutputStreamWriter
import java.net.HttpURLConnection
import java.net.URL

/**
 * Thin client for the control plane (backend/) — see docs/ARCHITECTURE.md.
 * Deliberately just HttpURLConnection + org.json (both in the Android SDK)
 * rather than OkHttp/Retrofit, to keep the dependency surface minimal for
 * this scaffold. Swap for Retrofit later if the API surface grows.
 *
 * Endpoints match backend/internal/api verbatim — see docs/BACKEND.md.
 */
class ApiClient(private val baseUrl: String) {

    data class Location(val id: String, val displayName: String)
    data class ConnectResult(val locationId: String, val config: String, val publicKey: String)

    /** POST /auth/register — anonymous, idempotent by device ID. */
    fun register(deviceId: String): String {
        val body = JSONObject().put("device_id", deviceId)
        val response = post("/auth/register", body, token = null)
        return response.getString("token")
    }

    /** GET /locations — requires a token from register(). */
    fun listLocations(token: String): List<Location> {
        val response = get("/locations", token)
        val arr = response.getJSONArray("locations")
        return (0 until arr.length()).map { i ->
            val obj = arr.getJSONObject(i)
            Location(obj.getString("id"), obj.getString("display_name"))
        }
    }

    /** POST /connect — requests a fresh WireGuard config for locationId. */
    fun connect(token: String, locationId: String): ConnectResult {
        val body = JSONObject().put("location_id", locationId)
        val response = post("/connect", body, token)
        return ConnectResult(
            locationId = response.getString("location_id"),
            config = response.getString("config"),
            publicKey = response.getString("public_key"),
        )
    }

    private fun get(path: String, token: String?): JSONObject {
        val conn = (URL(baseUrl + path).openConnection() as HttpURLConnection)
        conn.requestMethod = "GET"
        token?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
        return readResponse(conn)
    }

    private fun post(path: String, body: JSONObject, token: String?): JSONObject {
        val conn = (URL(baseUrl + path).openConnection() as HttpURLConnection)
        conn.requestMethod = "POST"
        conn.doOutput = true
        conn.setRequestProperty("Content-Type", "application/json")
        token?.let { conn.setRequestProperty("Authorization", "Bearer $it") }
        OutputStreamWriter(conn.outputStream).use { it.write(body.toString()) }
        return readResponse(conn)
    }

    private fun readResponse(conn: HttpURLConnection): JSONObject {
        val stream = if (conn.responseCode in 200..299) conn.inputStream else conn.errorStream
        val text = stream.bufferedReader().use { it.readText() }
        if (conn.responseCode !in 200..299) {
            throw ApiException(conn.responseCode, text)
        }
        return JSONObject(text)
    }

    class ApiException(val statusCode: Int, val body: String) :
        Exception("API error $statusCode: $body")

    companion object {
        /**
         * TODO: replace with the real deployed control-plane URL once a
         * central server exists (docs/OPEN_QUESTIONS.md). 10.0.2.2 is the
         * Android emulator's alias for the host machine's localhost, for
         * testing against `go run ./cmd/server` on your dev machine.
         */
        const val DEV_BASE_URL = "http://10.0.2.2:8090"
    }
}
