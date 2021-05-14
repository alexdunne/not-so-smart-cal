import axios from "axios";

interface GetEventWeatherRequestData {
  id: string;
}

interface GetEventWeatherResponse {
  data: {
    weather: {
      type: string;
      description: string;
      temp: string;
    };
  };
}

export const makeWeatherClient = () => {
  const client = axios.create({ baseURL: `${process.env.WEATHER_SERVICE}` });

  return {
    fetchEventWeather: async (data: GetEventWeatherRequestData) => {
      const response = await client.get<GetEventWeatherResponse>(`/event/${data.id}`);

      return response;
    },
  };
};
