import { ref, type Ref } from 'vue'
import { toast } from 'vue-sonner'
import { usePageSize } from './usePageSize'

export interface PaginatedListOptions<T, P = Record<string, any>> {
  pageSizeKey: string
  defaultPageSize?: number
  fetchApi: (params: P & { page: number; page_size: number }) => Promise<{ data?: T[] | null; total?: number } | null | undefined>
  errorMessage?: string
  onSuccess?: (data: T[], total: number) => void
}

export interface PaginatedListReturn<T, P = Record<string, any>> {
  list: Ref<T[]>
  total: Ref<number>
  page: Ref<number>
  pageSize: Ref<number>
  loading: Ref<boolean>
  loadData: (extraParams?: Partial<P>) => Promise<void>
  handlePageChange: (newPage: number, extraParams?: Partial<P>) => void
  handlePageSizeChange: (newSize: number, extraParams?: Partial<P>) => void
  debouncedSearch: (callback?: () => void, delay?: number) => void
  resetPage: () => void
}

export function usePaginatedList<T, P = Record<string, any>>(
  options: PaginatedListOptions<T, P>
): PaginatedListReturn<T, P> {
  const {
    pageSizeKey,
    defaultPageSize = 10,
    fetchApi,
    errorMessage = '加载数据失败',
    onSuccess
  } = options

  const { pageSize, handlePageSizeChange: persistPageSize } = usePageSize(pageSizeKey, defaultPageSize)

  const list = ref<T[]>([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(1)
  const loading = ref(false)

  let searchTimer: ReturnType<typeof setTimeout> | null = null

  async function loadData(extraParams?: Partial<P>) {
    loading.value = true
    try {
      const res = await fetchApi({
        page: page.value,
        page_size: pageSize.value,
        ...(extraParams as any)
      })
      list.value = Array.isArray(res?.data) ? res.data : []
      total.value = typeof res?.total === 'number' ? res.total : 0
      if (onSuccess) {
        onSuccess(list.value, total.value)
      }
    } catch (err: any) {
      list.value = []
      total.value = 0
      toast.error(`${errorMessage}: ${err?.message || '未知错误'}`)
    } finally {
      loading.value = false
    }
  }

  function handlePageChange(newPage: number, extraParams?: Partial<P>) {
    page.value = newPage
    loadData(extraParams)
  }

  function handlePageSizeChange(newSize: number, extraParams?: Partial<P>) {
    persistPageSize(newSize)
    page.value = 1
    loadData(extraParams)
  }

  function debouncedSearch(callback?: () => void, delay = 300) {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      page.value = 1
      if (callback) {
        callback()
      } else {
        loadData()
      }
    }, delay)
  }

  function resetPage() {
    page.value = 1
  }

  return {
    list,
    total,
    page,
    pageSize,
    loading,
    loadData,
    handlePageChange,
    handlePageSizeChange,
    debouncedSearch,
    resetPage
  }
}
