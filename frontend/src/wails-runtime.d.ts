declare module '@wailsio/runtime' {
  export const Events: {
    On: (eventName: string, callback: (event: any) => void) => void
    Emit: (eventName: string, data?: any) => void
  }
}
