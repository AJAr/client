// @flow
import {compose, connect, setDisplayName, type TypedState} from '../../util/container'
import SortBar from './sortbar'
import * as I from 'immutable'
import * as Types from '../../constants/types/fs'
import * as Constants from '../../constants/fs'
import * as FsGen from '../../actions/fs-gen'

type OwnProps = {
  path: Types.Path,
}

const mapStateToProps = (state: TypedState, {path}: OwnProps) => ({
  sortSetting: state.fs.pathUserSettings.get(path, Constants.makePathUserSetting()).get('sort'),
  _loadingPaths: state.fs.loadingPaths,
})

const mapDispatchToProps = (dispatch, {path}) => ({
  sortSettingToAction: (sortSetting: Types.SortSetting) => () => {
    dispatch(FsGen.createSortSetting({path, sortSetting}))
  },
})

const emptySet = I.Set()

const mergeProps = ({sortSetting, _loadingPaths}, {sortSettingToAction}, {path}: OwnProps) => ({
  sortSetting,
  folderIsPending: _loadingPaths.get(path, emptySet).size > 0,
  sortSettingToAction,
})

export default compose(
  connect(
    mapStateToProps,
    mapDispatchToProps,
    mergeProps
  ),
  setDisplayName('SortBar')
)(SortBar)
