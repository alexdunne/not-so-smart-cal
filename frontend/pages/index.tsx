import Head from "next/head";
import { gql, useQuery } from "@apollo/client";

import "react-big-calendar/lib/css/react-big-calendar.css";
import { Calendar, dateFnsLocalizer } from "react-big-calendar";
import format from "date-fns/format";
import parse from "date-fns/parse";
import parseISO from "date-fns/parseISO";
import startOfWeek from "date-fns/startOfWeek";
import getDay from "date-fns/getDay";

import { ClientOnly } from "../components/ClientOnly";
const locales = {
  "en-GB": require("date-fns/locale/en-GB"),
};

const localizer = dateFnsLocalizer({
  format,
  parse,
  startOfWeek,
  getDay,
  locales,
});

const GET_EVENTS_QUERY = gql`
  query ListEvents($input: EventsInput!) {
    events(input: $input) {
      id
      title
      location
      startsAt
      endsAt
      # weather {
      #   type
      #   description
      #   temp
      # }
    }
  }
`;

export default function Home() {
  return (
    <div>
      <Head>
        <title>Not So Smart Cal</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <ClientOnly>
        <HomeImpl />
      </ClientOnly>
    </div>
  );
}

const HomeImpl = () => {
  const { data, loading, error } = useQuery(GET_EVENTS_QUERY, {
    variables: {
      input: {
        startsAt: "2021-05-01T00:00:00.000Z",
        endsAt: "2021-05-31T00:00:00.000Z",
      },
    },
  });

  if (loading) {
    return <h2>Loading...</h2>;
  }

  if (error) {
    console.error(error);
    return null;
  }

  const events = data.events.map((event: any, index: number) => {
    return {
      id: index,
      title: event.title,
      start: parseISO(event.startsAt),
      end: parseISO(event.endsAt),
    };
  });

  return (
    <div>
      <Calendar localizer={localizer} events={events} startAccessor="start" endAccessor="end" style={{ height: 500 }} />
    </div>
  );
};
