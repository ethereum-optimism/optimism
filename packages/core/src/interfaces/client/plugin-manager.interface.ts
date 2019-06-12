import { PredicatePlugin } from './predicate-plugin.interface'

export interface PluginManager {
  /**
   * Loads the PredicatePlugin at the provided path and associates it with the provided address.
   *
   * @param address the address of the PredicatePlugin
   * @param path the path of the file of the PredicatePlugin
   * @returns the loaded PredicatePlugin
   */
  loadPlugin(address: string, path: string): Promise<PredicatePlugin>

  /**
   * Gets the PredicatePlugin associated with the provided address, if one exists.
   *
   * @param address the address of the PredicatePlugin
   * @returns the PredicatePlugin if one is associated with the provided Address, else undefined
   */
  getPlugin(address: string): Promise<PredicatePlugin | undefined>
}
