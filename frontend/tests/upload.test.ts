import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, onMounted, ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import Upload from '../app/pages/index.vue'
import CabinetUpload from '../app/pages/cabinets/[id].vue'

let wrapper: VueWrapper
let apiFetch: ReturnType<typeof vi.fn>

beforeEach(() => {
  vi.stubGlobal('ref', ref)
  vi.stubGlobal('computed', computed)
  vi.stubGlobal('onMounted', onMounted)
  vi.stubGlobal('definePageMeta', vi.fn())
  vi.stubGlobal('useRuntimeConfig', () => ({ public: { apiBase: '/api' } }))
  vi.stubGlobal('useRoute', () => ({ params: { id: 'cabinet' } }))
  apiFetch = vi.fn().mockImplementation(async (path: string) => ({
    ok: true,
    json: async () => path === '/upload' ? { id: 'test', fileName: 'test.xlsx', rows: 1 } : [],
  }))
  vi.stubGlobal('useAuth', () => ({ apiFetch, token: ref('test') }))
})
afterEach(() => { wrapper?.unmount(); vi.unstubAllGlobals() })

describe('upload report type', () => {
  for (const [name, component] of [['home', Upload], ['cabinet', CabinetUpload]] as const) {
    it.each(['hr', 'avito'])(`${name} submits selected %s type`, async type => {
      wrapper = mount(component, { global: { stubs: { NuxtLink: { template: '<a><slot /></a>' } } } })
      await flushPromises()
      await wrapper.find('select').setValue(type)
      const input = wrapper.find('input[type="file"]')
      Object.defineProperty(input.element, 'files', { value: [new File(['fixture'], 'test.xlsx')] })
      await input.trigger('change')
      await flushPromises()
      const call = apiFetch.mock.calls.find(([path]) => path === '/upload')
      expect(call).toBeDefined()
      expect(call![1].body.get('type')).toBe(type)
      if (name === 'cabinet') expect(call![1].body.get('cabinetId')).toBe('cabinet')
    })
  }
})
