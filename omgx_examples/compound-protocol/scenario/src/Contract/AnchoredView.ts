import {Contract} from '../Contract';
import {Callable} from '../Invokation';

interface AnchoredViewMethods {
  getUnderlyingPrice(asset: string): Callable<number>
}

export interface AnchoredView extends Contract {
  methods: AnchoredViewMethods
}
