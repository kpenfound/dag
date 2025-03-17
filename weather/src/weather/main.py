import requests
from typing import Annotated
from dagger import Doc, function, object_type


@object_type
class Weather:
    @function
    def City(
        self,
        name: Annotated[str, Doc("Name of a city")]
    ) -> str:
        """Returns the current weather in a city"""
        coordinates = get_coordinates(name)
        weather = get_weather(coordinates['latitude'], coordinates['longitude'])
        return f"{weather}°C"

    @function
    def Coordinates(
        self,
        latitude: Annotated[str, Doc("latitude of a location")],
        longitude: Annotated[str, Doc("longitude of a location")]
    ) -> str:
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
