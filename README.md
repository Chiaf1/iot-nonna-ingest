# iot-nonna-ingest
Questa repo fa parte del progetto iot-nonna, in particolare questa repo contiene il servizio di lettura topic mqtt per inserimento valori nel db. La base del programma che fa mqtt ingestion è clonata dalla repo mqttingester. Poi viene aggiunta tutta la parte di interfaccia con il db postgres e l'handling dei topic dinamicamente.

## Handling dinamico dei topic mqtt:
L'esigenza nasce dalla possibilità di aggiungere sensori e dispositivi di campionamento (e di conseguenza dei loro topic) con libertà durante il funzionamento del sistema.
Per farlo allora è necessario utilizzare una struttura ben definita all'interno del db dove per ogni topic viene definita la tabella di destinazione e la colonne da riempire con l'associata tag del payload mqtt.
All'interno del programma la funzione dedicata all'handling dei messaggi mqtt avrà accesso ai topic con i loro metadati. Di conseguenza interpreterà i dati e li inserirà nel db.

L'handling dei messaggi è fatto a workerpool quindi è fondamentale prestare attenzione alla struttura centrale che imagazzina topic e i loro metadati. Questa struttura deve essere threadsafe e periodicamente deve essere aggiornata interrogando il db per eventuali cambiamenti. Ad ogni cambiamento di questa struttura il programma dovrà anche gestire il subscribe dei nuovi topic e l'unsubscribe dei topci non più esistenti.

# Stato repo:
Ancora in fase di studio preliminare