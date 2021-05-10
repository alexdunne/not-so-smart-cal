import axios from "axios";
import { objectType, queryField } from "nexus";

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