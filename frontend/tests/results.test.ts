import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, onMounted, ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import Results from '../app/pages/results.vue'

const countLabel = 'Количество объявлений'
const row = (key: string, listingCount: number, shows = 20) => ({ key, listingCount, shows, contacts: 4 })
const summary = { totalShows: 20, totalViews: 8, totalContacts: 4, topCities: [] }
const report = (reportType: string, reportId: string) => ({ reportType, reportId, fileName: reportId, summary, stats: [row('Анна', 2), row('Борис', 12)] })
let wrapper: VueWrapper
let apiFetch: ReturnType<typeof vi.fn>

beforeEach(() => {
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('computed', computed)
  vi.stubGlobal('onMounted', onMounted)
  vi.stubGlobal('definePageMeta', vi.fn())
  vi.stubGlobal('useRuntimeConfig', () => ({ public: { apiBase: '/api' } }))
  vi.stubGlobal('useRoute', () => ({ query: { ids: 'a,b' } }))
  apiFetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ reports: [report('hr', 'HR'), report('regular', 'Regular'), report('', 'Legacy')] }) })
  vi.stubGlobal('useAuth', () => ({ apiFetch, token: ref('test') }))
})
afterEach(() => { wrapper?.unmount(); vi.unstubAllGlobals() })

async function openResults() {
  wrapper = mount(Results, { global: { stubs: { NuxtLink: { template: '<a><slot /></a>' } } } })
  await flushPromises()
}
async function group(label: string) {
  await wrapper.findAll('button').find(b => b.text() === label)!.trigger('click')
  await flushPromises()
}

describe('HR grouped listing count', () => {
  it('renders employee-object counts left of Shows in every employee section', async () => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({ reports: [{
      ...report('hr', 'HR'), stats: [
        { ...row('A', 2), employee: 'Анна', object: 'Офис' },
        { ...row('B', 12), employee: 'Борис', object: 'Склад' },
      ],
    }] }) })
    await openResults()
    await group('Объекты сотрудников')
    expect(wrapper.findAll('section h3').map(h => h.text())).toEqual(['Сотрудник: Анна', 'Сотрудник: Борис'])
    const tables = wrapper.findAll('table')
    expect(tables).toHaveLength(2)
    for (const [i, table] of tables.entries()) {
      expect(table.findAll('th').slice(0, 3).map(h => h.text())).toEqual(['Объект', countLabel, 'Показы'])
      const cells = table.findAll('tbody tr')[0]!.findAll('td')
      expect(cells[1]!.text()).toBe(['2', '12'][i])
      expect(cells[2]!.text()).toBe('20')
      expect(cells).toHaveLength(table.findAll('th').length)
    }
  })
  it.each([['hr', 'regular'], ['regular'], [''], []])('hides HR grouping for non-all-HR reports %j', async (...types) => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({ reports: types.map((type, i) => report(type as string, String(i))) }) })
    await openResults()
    for (const label of ['По сотрудникам', 'По объектам', 'Объекты сотрудников']) {
      expect(wrapper.findAll('button').some(b => b.text() === label)).toBe(false)
    }
    expect(wrapper.text()).not.toContain(countLabel)
  })
  it.each(['По сотрудникам', 'По объектам'])('renders comparison counts and aligned P1/P2/Δ for %s', async label => {
    vi.stubGlobal('useRoute', () => ({ query: { ids: 'a,b', compare: '1' } }))
    const early = [row('Анна', 2), row('Ушёл', 1), row('Новый', 0)]
    const late = [row('Анна', 3), row('Ушёл', 0), row('Новый', 2)]
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({
      reportTypes: ['hr', 'hr'],
      early: { reportType: 'hr', stats: early }, late: { reportType: 'hr', stats: late },
      delta: [row('Анна', 1), row('Ушёл', -1), row('Новый', 2)],
    }) })
    await openResults()
    await group(label)
    const table = wrapper.find('table')
    const headings = table.findAll('thead tr')[0]!.findAll('th')
    expect(headings[1]!.text()).toBe(countLabel)
    expect(headings[1]!.attributes('colspan')).toBe('3')
    expect(headings[2]!.text()).toBe('Показы')
    const subheadings = table.findAll('thead tr')[1]!.findAll('th')
    expect(subheadings.slice(1, 4).map(c => c.text())).toEqual(['П1', 'П2', 'Δ'])
    const rows = table.findAll('tbody tr')
    expect(rows.map(r => r.findAll('td').slice(1, 4).map(c => c.text()))).toEqual([['2', '3', '+1'], ['1', '0', '-1'], ['0', '2', '+2']])
    for (const r of rows) expect(r.findAll('td')).toHaveLength(subheadings.length)
    expect(subheadings).toHaveLength(37)
    await group('По городам')
    expect(wrapper.find('table').text()).not.toContain(countLabel)
    expect(wrapper.findAll('thead tr')[1]!.findAll('th')).toHaveLength(34)
  })
  it.each([['hr', 'regular'], ['regular', 'hr'], ['regular', 'regular'], ['', '']])('does not add comparison count for %s/%s', async (earlyType, lateType) => {
    vi.stubGlobal('useRoute', () => ({ query: { ids: 'a,b', compare: '1' } }))
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({
      reportTypes: [earlyType, lateType],
      early: { reportType: earlyType, stats: [row('Анна', 2)] },
      late: { reportType: lateType, stats: [row('Анна', 3)] }, delta: [row('Анна', 1)],
    }) })
    await openResults()
    for (const label of ['По сотрудникам', 'По объектам']) {
      expect(wrapper.findAll('button').some(b => b.text() === label)).toBe(false)
      expect(wrapper.find('table').text()).not.toContain(countLabel)
      expect(wrapper.findAll('thead tr')[1]!.findAll('th')).toHaveLength(34)
    }
  })
  it.each(['По сотрудникам', 'По объектам'])('renders and sorts only HR %s', async label => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({ reports: [report('hr', 'HR')] }) })
    await openResults()
    await group(label)
    const tables = wrapper.findAll('table')
    const headers = tables[0]!.findAll('th')
    expect(headers[1]!.text()).toBe(countLabel)
    expect(headers[2]!.text()).toBe('Показы')
    expect(tables[0]!.findAll('tbody tr')[0]!.findAll('td').slice(0, 3).map(c => c.text())).toEqual(['Анна', '2', '20'])
    await headers[1]!.trigger('click')
    expect(tables[0]!.findAll('tbody tr').map(r => r.findAll('td')[1]!.text())).toEqual(['12', '2'])
    await headers[1]!.trigger('click')
    expect(tables[0]!.findAll('tbody tr').map(r => r.findAll('td')[1]!.text())).toEqual(['2', '12'])
    for (const table of tables) {
      expect(table.findAll('tbody tr')[0]!.findAll('td')).toHaveLength(table.findAll('th').length)
    }
  })
  it.each(['По городам', 'По категориям', 'По подкатегориям', 'По объявлениям'])('leaves %s columns unchanged', async label => {
    await openResults()
    await group(label)
    for (const table of wrapper.findAll('table')) {
      expect(table.text()).not.toContain(countLabel)
      const headers = table.findAll('th')
      expect(headers).toHaveLength(label === 'По объявлениям' ? 13 : 12)
      expect(headers[label === 'По объявлениям' ? 2 : 1]!.text()).toBe('Показы')
    }
  })
})
