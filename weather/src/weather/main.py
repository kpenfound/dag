import requests
from dagger import function, object_type


@object_type
class Weather:
    @function
    def City(self, name: str) -> str:
        """Returns the current weather in a city"""
        coordinates = get_coordinates(name)
        weather = get_weather(coordinates['latitude'], coordinates['longitude'])
        return f"{weather}°C"

    @function
    def Coordinates(self, latitude: str, longitude: str) -> str:
        """Returns the current weather at the given coordinates"""
        weather = get_weather(latitude, longitude)
        return f"{weather}°C"


def get_weather(latitude, longitude):
    response = requests.get(f"https://api.open-meteo.com/v1/forecast?latitude={latitude}&longitude={longitude}&current=temperature_2m,wind_speed_10m&hourly=temperature_2m,relative_humidity_2m,wind_speed_10m")
    data = response.json()
    return data['current']['temperature_2m']

def get_coordinates(search):
    response = requests.get(f"https://geocoding-api.open-meteo.com/v1/search?name={search}&count=1&format=json")
    data = response.json()
    return data['results'][0]
