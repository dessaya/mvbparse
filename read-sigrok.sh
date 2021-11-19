# conectar D0 al input
# ff => high
# fe => low
sigrok-cli -d fx2lafw --continuous --config samplerate=12m --channels D0 -O binary | python3 mvb_signal.py | python3 mvbparse.py
