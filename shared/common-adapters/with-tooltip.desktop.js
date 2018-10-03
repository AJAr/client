// @flow
import * as React from 'react'
import * as Styles from '../styles'
import Box from './box'
import Toast from './toast'
import Text from './text'
import {type Props} from './with-tooltip'

type State = {
  mouseIn: boolean,
}

class WithTooltip extends React.Component<Props, State> {
  state = {
    mouseIn: false,
  }
  _attachmentRef: ?React.Component<any> = null
  _onMouseEnter = () => {
    this.setState({mouseIn: true})
  }
  _onMouseLeave = () => {
    this.setState({mouseIn: false})
  }
  _setAttachmentRef = attachmentRef => (this._attachmentRef = attachmentRef)
  render() {
    return (
      <Box
        style={this.props.containerStyle}
        ref={this._setAttachmentRef}
        onMouseEnter={this._onMouseEnter}
        onMouseLeave={this._onMouseLeave}
      >
        {this.props.children}
        <Toast
          containerStyle={this.props.multiline ? styles.containerMultiline : styles.container}
          visible={this.state.mouseIn}
          attachTo={() => this._attachmentRef}
          position={this.props.position || 'top center'}
        >
          <Text type="BodySmall" style={styles.text}>
            {this.props.text}
          </Text>
        </Toast>
      </Box>
    )
  }
}

const styles = Styles.styleSheetCreate({
  container: {
    borderRadius: 20,
  },
  containerMultiline: {
    borderRadius: 4,
    width: 320,
    minWidth: 320,
    maxWidth: 320,
  },
  text: Styles.platformStyles({
    isElectron: {
      color: Styles.globalColors.white,
      wordBreak: 'break-all',
    },
  }),
})

export default WithTooltip
