import axios from "axios";
import { objectType, queryField, asNexusMethod } from "nexus";
import {  DateTimeResolver } from 'graphql-scalars';

export const DateTime = DateTimeResolver

export const CalendarServiceDiagnosticsType = objectType({
  name: 'CalendarServiceDiagnostics',
  definition(t) {
    t.string('version')
  }
})

export const DiagnosticsType = objectType({
  name: 'Diagnostics',
  definition(t) {
    t.field('calendar', {
      type: CalendarServiceDiagnosticsType,
    })
  },
})

export const DiagnosticsQuery = queryField('diagnostics', {
  type: DiagnosticsType,
  async resolve() {
    const response = await axios.get(`${process.env.CALENDAR_SERVICE}`);

    return {
      calendar: response.data
    }
  },
})

export const EventType = objectType({
  name: "Event",
  definition(t) {
    t.string('id')
    t.string('title')
    t.string('location')
    t.field('startsAt', {
      type: 'DateTime'
    })
    t.field('endsAt', {
      type: 'DateTime'
    })
  }
})

export * from "./CreateEventMutation"
