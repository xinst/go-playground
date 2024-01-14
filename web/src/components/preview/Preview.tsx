import React, {useMemo, useEffect, useState, useRef} from 'react';
import {MessageBar, MessageBarType, useTheme} from '@fluentui/react';

import {getDefaultFontFamily} from '~/services/fonts';
import {connect, StatusState} from '~/store';

import { XTerm } from '~/components/utils/XTerm';
import { formatEvalEvent } from './utils';

import './Preview.css';

interface OwnProps {}

interface StateProps {
  status?: StatusState
}

interface PreviewContentProps {
  status?: StatusState
}

const PreviewContent: React.FC<PreviewContentProps> = ({status}) => {
  const [offset, setOffset] = useState(0);
  const xtermRef = useRef<XTerm>(null);

  const isClean = !status || !status?.dirty;
  const isRunning = status?.running;
  const events = status?.events;

  const terminal = xtermRef.current?.terminal;
  useEffect(() => {
    if (!events?.length) {
      setOffset(0);
      terminal?.clear();
      return;
    }
    if (offset === 0) {
      terminal?.clear();
    }

    const batch = events?.slice(offset);
    if (!batch) {
      return;
    }

    batch.map(formatEvalEvent).forEach((msg) => terminal?.write(msg));
    setOffset(offset + batch.length);
  }, [terminal, offset, events ])

  useEffect(() => {
    if (isClean) {
      setOffset(0)
    }

  }, [isClean])

  if (status?.lastError) {
    return (
      <MessageBar messageBarType={MessageBarType.error} isMultiline={true}>
        <b className='app-preview__label'>Error</b>
        <pre className='app-preview__errors'>
            {status.lastError}
          </pre>
      </MessageBar>
    );
  }

  return (
    <>
      {
        (
          isClean ? (
            <span>Press "Run" to compile program.</span>
          ) : (
            <XTerm
              ref={xtermRef}
              options={{convertEol: true}}
            />
          )
        )
      }
    </>
  );
}

const Preview: React.FC<StateProps & OwnProps> = ({ status }) => {
  const theme = useTheme();
  const styles = useMemo(() => {
    const { palette } = theme;
    return {
      backgroundColor: palette.neutralLight,
      color: palette.neutralDark,
      fontFamily: getDefaultFontFamily(),
    }
  }, [theme]);

  return (
    <div className="app-preview" style={styles}>
      <div className='app-preview__content'>
        <PreviewContent status={status} />
      </div>
    </div>
  )
}

const ConnectedPreview = connect<StateProps, OwnProps>((
  { status }
  // { settings: {darkMode}, runTarget: { target }, status }
) => ({
  status
}))(Preview);

export default ConnectedPreview;
